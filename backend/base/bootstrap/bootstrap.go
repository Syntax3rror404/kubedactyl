// Package bootstrap prepares the cluster when the panel starts: the CRDs, the panel namespace, the session
// signing key and the first administrator. The app runs its own steps after Run with the same client.
package bootstrap

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"app/api/v1alpha1"
	"app/app"
	"app/base/auth"
)

// Scheme knows the Kubernetes types, the app's CRDs and the CRD type itself (installCRDs reads it back).
func Scheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme, v1alpha1.AddToScheme, apiextensionsv1.AddToScheme,
	} {
		if err := add(scheme); err != nil {
			return nil, fmt.Errorf("building scheme: %w", err)
		}
	}
	return scheme, nil
}

// Result is what the start-up steps give the API.
type Result struct {
	Signer *auth.Signer
	// SetupToken must accompany the setup request; "" once an administrator exists.
	SetupToken string
}

// Run prepares the cluster with a direct client (the cache does not run yet). url is the address of the web
// interface, for the setup link in the log.
//  1. CRDs: the embedded manifests (crds) installed or updated, so the types the panel uses exist.
//  2. Namespace: the panel namespace.
//  3. Session key: loaded or created, so sessions survive restarts.
//  4. First admin: created from the environment, or a setup token when there is none.
func Run(ctx context.Context, c client.Client, crds fs.FS, namespace, url string, log *slog.Logger) (*Result, error) {
	// 1. CRDs
	if err := installCRDs(ctx, c, crds); err != nil {
		return nil, fmt.Errorf("installing CRDs: %w", err)
	}
	// 2. Namespace
	if err := ensureNamespace(ctx, c, namespace); err != nil {
		return nil, fmt.Errorf("creating namespace: %w", err)
	}
	// 3. Session key
	signer, err := loadSessionSigner(ctx, c, namespace)
	if err != nil {
		return nil, fmt.Errorf("loading session key: %w", err)
	}
	// 4. First admin
	token, err := createAdmin(ctx, c, namespace, url, log)
	if err != nil {
		return nil, fmt.Errorf("creating initial admin: %w", err)
	}
	return &Result{Signer: signer, SetupToken: token}, nil
}

// managedBy marks the objects the panel created.
func managedBy() map[string]string { return map[string]string{"app.kubernetes.io/managed-by": app.ID} }

// installCRDs applies the embedded CRDs with server-side apply and waits until they are established.
func installCRDs(ctx context.Context, c client.Client, crds fs.FS) error {
	names, err := applyCRDs(ctx, c, crds)
	if err != nil {
		return err
	}
	return waitEstablished(ctx, c, names)
}

// applyCRDs server-side applies the CRD manifests and returns their names.
func applyCRDs(ctx context.Context, c client.Client, crds fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(crds, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		data, err := fs.ReadFile(crds, e.Name())
		if err != nil {
			return nil, err
		}
		u := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(data, &u.Object); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if err := c.Apply(
			ctx, client.ApplyConfigurationFromUnstructured(u), client.FieldOwner(app.ID), client.ForceOwnership,
		); err != nil {
			return nil, fmt.Errorf("apply %s: %w", u.GetName(), err)
		}
		names = append(names, u.GetName())
	}
	return names, nil
}

// waitEstablished waits (up to a minute) until the API server serves every CRD.
func waitEstablished(ctx context.Context, c client.Client, names []string) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		for _, n := range names {
			obj := &apiextensionsv1.CustomResourceDefinition{}
			if err := c.Get(ctx, client.ObjectKey{Name: n}, obj); err != nil || !crdEstablished(obj) {
				return false, nil
			}
		}
		return true, nil
	})
}

func crdEstablished(obj *apiextensionsv1.CustomResourceDefinition) bool {
	for _, cond := range obj.Status.Conditions {
		if cond.Type == apiextensionsv1.Established {
			return cond.Status == apiextensionsv1.ConditionTrue
		}
	}
	return false
}

// ensureNamespace creates the panel namespace if needed.
func ensureNamespace(ctx context.Context, c client.Client, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: managedBy()}}
	if err := c.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// sessionKeySecret holds the key that signs session tokens.
var sessionKeySecret = app.ID + "-auth"

// loadSessionSigner reads (or creates) the random key that signs session tokens.
func loadSessionSigner(ctx context.Context, c client.Client, ns string) (*auth.Signer, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: sessionKeySecret}, secret)
	if apierrors.IsNotFound(err) {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: sessionKeySecret, Namespace: ns, Labels: managedBy()},
			Data:       map[string][]byte{"session-key": key},
		}
		// Two panels starting at once: the one that loses the race uses the stored key.
		switch err := c.Create(ctx, secret); {
		case apierrors.IsAlreadyExists(err):
			return loadSessionSigner(ctx, c, ns)
		case err != nil:
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return auth.NewSigner(secret.Data["session-key"])
}

// The environment variables of the first administrator (KUBEDACTYL_ADMIN_PASSWORD, KUBEDACTYL_SETUP_TOKEN).
var (
	adminPasswordEnv = app.EnvPrefix + "_ADMIN_PASSWORD"
	setupTokenEnv    = app.EnvPrefix + "_SETUP_TOKEN"
)

// createAdmin creates the user "admin" from <PREFIX>_ADMIN_PASSWORD when no active administrator exists (for
// automated installations). Without the variable, or when a user "admin" exists but is no active administrator,
// the web UI shows the setup page, which needs the setup token returned here (from <PREFIX>_SETUP_TOKEN or
// generated and logged once). The token is empty when an administrator exists.
func createAdmin(ctx context.Context, c client.Client, ns, url string, log *slog.Logger) (string, error) {
	var users v1alpha1.UserList
	if err := c.List(ctx, &users, client.InNamespace(ns)); err != nil {
		return "", err
	}
	for _, u := range users.Items {
		if u.Spec.Role == v1alpha1.RoleAdmin && !u.Spec.Disabled {
			return "", nil
		}
	}
	password := os.Getenv(adminPasswordEnv)
	if password == "" {
		return setupToken(url, log), nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("%s: %w", adminPasswordEnv, err)
	}
	admin := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: "admin", Namespace: ns},
		Spec:       v1alpha1.UserSpec{DisplayName: "Administrator", Role: v1alpha1.RoleAdmin, PasswordHash: hash},
	}
	// Fail-safe: an existing "admin" (disabled or demoted) is never changed; the setup page creates another one.
	switch err := c.Create(ctx, admin); {
	case apierrors.IsAlreadyExists(err):
		log.Warn(adminPasswordEnv + " ignored: the user admin exists but is no active administrator")
		return setupToken(url, log), nil
	case err != nil:
		return "", err
	}
	log.Info("created initial admin user from "+adminPasswordEnv, "user", "admin")
	return "", nil
}

// setupToken returns the token of the setup page: <PREFIX>_SETUP_TOKEN, or a generated one that is logged once.
func setupToken(url string, log *slog.Logger) string {
	token := os.Getenv(setupTokenEnv)
	if token == "" {
		token = auth.NewSetupToken()
		log.Warn("no administrator yet, open the setup page with this one-time link", "url", url+"/setup?token="+token)
	} else {
		log.Warn("no administrator yet, open the setup page with "+setupTokenEnv, "url", url+"/setup")
	}
	return token
}
