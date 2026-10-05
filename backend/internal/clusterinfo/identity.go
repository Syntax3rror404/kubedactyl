package clusterinfo

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// CertInfo summarizes an X.509 certificate.
type CertInfo struct {
	Subject   string    `json:"subject"   example:"CN=kubernetes"`
	Issuer    string    `json:"issuer"    example:"CN=kubernetes"`
	NotBefore time.Time `json:"notBefore"`
	NotAfter  time.Time `json:"notAfter"`
}

// Kubeconfig describes the kubeconfig entry the panel uses (no secrets).
type Kubeconfig struct {
	Files     []string `json:"files"               example:"/home/user/.kube/config"`
	Context   string   `json:"context"             example:"my-cluster"`
	Cluster   string   `json:"cluster"             example:"my-cluster"`
	User      string   `json:"user"                example:"oidc-user"`
	Namespace string   `json:"namespace,omitempty"`
	// Contexts lists the names of all contexts in the kubeconfig.
	Contexts []string `json:"contexts"`
}

// ServiceAccount describes the in-cluster identity.
type ServiceAccount struct {
	Namespace      string     `json:"namespace"                example:"kubedactyl"`
	Name           string     `json:"name"                     example:"kubedactyl"`
	TokenExpiresAt *time.Time `json:"tokenExpiresAt,omitempty"`
	Audiences      []string   `json:"audiences,omitempty"`
}

// Auth describes how the panel authenticates (never contains secrets).
type Auth struct {
	// Method: client-certificate, token, exec, auth-provider, basic, service-account or none.
	Method            string    `json:"method"                      example:"exec"`
	ExecCommand       string    `json:"execCommand,omitempty"       example:"kubectl"`
	ExecArgs          []string  `json:"execArgs,omitempty"          example:"oidc-login,get-token"`
	ExecAPIVersion    string    `json:"execApiVersion,omitempty"`
	AuthProvider      string    `json:"authProvider,omitempty"`
	TokenPreview      string    `json:"tokenPreview,omitempty"      example:"eyJh…Qw"`
	ClientCertificate *CertInfo `json:"clientCertificate,omitempty"`
	Impersonating     string    `json:"impersonating,omitempty"`
}

// TLS describes the transport security towards the API server.
type TLS struct {
	Insecure   bool      `json:"insecure"`
	ServerName string    `json:"serverName,omitempty"`
	CA         *CertInfo `json:"ca,omitempty"`
	// CASource is "kubeconfig", "file", "system" (no CA configured) or "service-account".
	CASource string `json:"caSource" example:"kubeconfig"`
}

// Subject is who the API server says the panel is (SelfSubjectReview).
type Subject struct {
	Username string              `json:"username"        example:"oidc:alice"`
	UID      string              `json:"uid,omitempty"`
	Groups   []string            `json:"groups"          example:"oidc:kube-admins"`
	Extra    map[string][]string `json:"extra,omitempty"`
}

// Permission is one access check the panel depends on.
type Permission struct {
	Label     string `json:"label"               example:"Exec into pods"`
	Verb      string `json:"verb"                example:"create"`
	Group     string `json:"group,omitempty"     example:"metrics.k8s.io"`
	Resource  string `json:"resource"            example:"pods/exec"`
	Namespace string `json:"namespace,omitempty"`
	Allowed   bool   `json:"allowed"`
}

// Identity is the connection summary of the panel.
type Identity struct {
	// Mode is "kubeconfig" or "in-cluster".
	Mode           string          `json:"mode"                     example:"kubeconfig"`
	Server         string          `json:"server"                   example:"https://k8s.example.com:6443"`
	Version        string          `json:"version"                  example:"v1.36.4"`
	Platform       string          `json:"platform"                 example:"linux/amd64"`
	Protocol       string          `json:"protocol"                 example:"HTTP/1.1"`
	Kubeconfig     *Kubeconfig     `json:"kubeconfig,omitempty"`
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`
	Auth           Auth            `json:"auth"`
	TLS            TLS             `json:"tls"`
	Subject        *Subject        `json:"subject,omitempty"`
	SubjectError   string          `json:"subjectError,omitempty"`
	Permissions    []Permission    `json:"permissions"`
}

const saDir = "/var/run/secrets/kubernetes.io/serviceaccount"

// inCluster mirrors controller-runtime: without KUBECONFIG, the in-cluster config wins.
func (s *Service) inCluster() bool {
	return s.KubeContext == "" && os.Getenv("KUBECONFIG") == "" && os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// Identity builds the connection summary.
func (s *Service) Identity(ctx context.Context) *Identity {
	id := &Identity{Server: s.Config.Host, Protocol: "HTTP/2"}
	if s.HTTP1 {
		id.Protocol = "HTTP/1.1"
	}
	if v, err := s.Kube.Clientset.Discovery().ServerVersion(); err == nil {
		id.Version, id.Platform = v.GitVersion, v.Platform
	}
	id.TLS = TLS{Insecure: s.Config.Insecure, ServerName: s.Config.ServerName, CASource: "system"}
	switch {
	case len(s.Config.CAData) > 0:
		id.TLS.CASource, id.TLS.CA = "kubeconfig", parseCert(s.Config.CAData)
	case s.Config.CAFile != "":
		id.TLS.CASource = "file"
		if data, err := os.ReadFile(s.Config.CAFile); err == nil {
			id.TLS.CA = parseCert(data)
		}
	}

	if s.inCluster() {
		id.Mode = "in-cluster"
		id.Auth = Auth{Method: "service-account"}
		id.TLS.CASource = "service-account"
		id.ServiceAccount = serviceAccountInfo()
	} else {
		id.Mode = "kubeconfig"
		kc, auth := s.kubeconfigInfo()
		id.Kubeconfig, id.Auth = kc, auth
	}
	if s.Config.Impersonate.UserName != "" {
		id.Auth.Impersonating = s.Config.Impersonate.UserName
	}

	reviews := s.Kube.Clientset.AuthenticationV1().SelfSubjectReviews()
	review, err := reviews.Create(ctx, &authnv1.SelfSubjectReview{}, metav1.CreateOptions{})
	if err != nil {
		id.SubjectError = err.Error()
	} else {
		u := review.Status.UserInfo
		id.Subject = &Subject{Username: u.Username, UID: u.UID, Groups: u.Groups, Extra: map[string][]string{}}
		for k, v := range u.Extra {
			if !strings.Contains(strings.ToLower(k), "credential") { // credential ids are not interesting
				id.Subject.Extra[k] = v
			}
		}
	}
	id.Permissions = s.permissions(ctx)
	return id
}

// secretArg matches exec arguments that carry credentials.
var secretArg = regexp.MustCompile(`(?i)(secret|token|password|passwd|key)`)

// redactArgs hides values of arguments that look like credentials.
func redactArgs(args []string) []string {
	out := make([]string, len(args))
	hideNext := false
	for i, a := range args {
		switch {
		case hideNext:
			out[i], hideNext = "••••", false
		case strings.HasPrefix(a, "-") && secretArg.MatchString(a):
			if k, _, ok := strings.Cut(a, "="); ok {
				out[i] = k + "=••••"
			} else {
				out[i], hideNext = a, true
			}
		default:
			out[i] = a
		}
	}
	return out
}

func (s *Service) kubeconfigInfo() (*Kubeconfig, Auth) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: s.KubeContext}
	raw, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).RawConfig()
	kc := &Kubeconfig{Files: rules.GetLoadingPrecedence(), Contexts: []string{}}
	if err != nil {
		return kc, Auth{Method: "none"}
	}
	return summarizeKubeconfig(raw, s.KubeContext, kc.Files)
}

// summarizeKubeconfig extracts the active context and its authentication without secrets.
func summarizeKubeconfig(raw clientcmdapi.Config, override string, files []string) (*Kubeconfig, Auth) {
	kc := &Kubeconfig{Files: files, Contexts: []string{}, Context: raw.CurrentContext}
	if override != "" {
		kc.Context = override
	}
	for name := range raw.Contexts {
		kc.Contexts = append(kc.Contexts, name)
	}
	sort.Strings(kc.Contexts)
	auth := Auth{Method: "none"}
	ctx := raw.Contexts[kc.Context]
	if ctx == nil {
		return kc, auth
	}
	kc.Cluster, kc.User, kc.Namespace = ctx.Cluster, ctx.AuthInfo, ctx.Namespace
	ai := raw.AuthInfos[ctx.AuthInfo]
	if ai == nil {
		return kc, auth
	}
	switch {
	case ai.Exec != nil:
		auth.Method, auth.ExecCommand, auth.ExecArgs, auth.ExecAPIVersion = "exec", ai.Exec.Command, redactArgs(
			ai.Exec.Args,
		), ai.Exec.APIVersion
	case ai.AuthProvider != nil:
		auth.Method, auth.AuthProvider = "auth-provider", ai.AuthProvider.Name
	case len(ai.ClientCertificateData) > 0 || ai.ClientCertificate != "":
		auth.Method = "client-certificate"
		data := ai.ClientCertificateData
		if len(data) == 0 {
			data, _ = os.ReadFile(ai.ClientCertificate)
		}
		auth.ClientCertificate = parseCert(data)
	case ai.Token != "" || ai.TokenFile != "":
		auth.Method = "token"
		if ai.Token != "" {
			auth.TokenPreview = preview(ai.Token)
		}
	case ai.Username != "":
		auth.Method = "basic"
	}
	if ai.Impersonate != "" {
		auth.Impersonating = ai.Impersonate
	}
	return kc, auth
}

func preview(token string) string {
	if len(token) <= 10 {
		return "••••"
	}
	return token[:4] + "…" + token[len(token)-2:]
}

func parseCert(data []byte) *CertInfo {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	return &CertInfo{
		Subject:   cert.Subject.String(),
		Issuer:    cert.Issuer.String(),
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
	}
}

// serviceAccountInfo reads the mounted service account token (payload only, not verified).
func serviceAccountInfo() *ServiceAccount {
	sa := &ServiceAccount{}
	if ns, err := os.ReadFile(saDir + "/namespace"); err == nil {
		sa.Namespace = strings.TrimSpace(string(ns))
	}
	token, err := os.ReadFile(saDir + "/token")
	if err != nil {
		return sa
	}
	parts := strings.Split(string(token), ".")
	if len(parts) < 2 {
		return sa
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return sa
	}
	var claims struct {
		Sub string   `json:"sub"`
		Exp int64    `json:"exp"`
		Aud []string `json:"aud"`
	}
	if json.Unmarshal(payload, &claims) == nil {
		if name, ok := strings.CutPrefix(claims.Sub, "system:serviceaccount:"+sa.Namespace+":"); ok {
			sa.Name = name
		}
		if claims.Exp > 0 {
			t := time.Unix(claims.Exp, 0).UTC()
			sa.TokenExpiresAt = &t
		}
		sa.Audiences = claims.Aud
	}
	return sa
}
