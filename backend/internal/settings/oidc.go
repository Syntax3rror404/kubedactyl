package settings

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/kube"
	"app/internal/sso"
	"app/internal/tenancy"
	"app/internal/validation"
)

// OIDCSecret holds the client secret of the identity provider (key client-secret) in the panel namespace: it
// is kept out of the settings, which every user can read.
const (
	OIDCSecret    = "kubedactyl-oidc"
	oidcSecretKey = "client-secret"
	maxOIDCName   = 40
)

// normalizeOIDC trims the single sign-on settings and, while it is on, requires the client and both groups and
// checks that the issuer answers.
func normalizeOIDC(ctx context.Context, o v1alpha1.OIDCSettings) (v1alpha1.OIDCSettings, error) {
	for _, f := range []*string{
		&o.Name, &o.IssuerURL, &o.ClientID, &o.RedirectURL, &o.UsernameClaim, &o.GroupsClaim, &o.AdminGroup,
		&o.UserGroup,
	} {
		*f = strings.TrimSpace(*f)
	}
	if n := len([]rune(o.Name)); n > maxOIDCName {
		return o, validation.Field("oidc.name", fmt.Errorf("too long (%d of %d characters)", n, maxOIDCName))
	}
	if !o.Enabled {
		return o, nil
	}
	for _, f := range []textField{
		{"oidc.issuerUrl", o.IssuerURL}, {"oidc.clientId", o.ClientID}, {"oidc.redirectUrl", o.RedirectURL},
		{"oidc.adminGroup", o.AdminGroup}, {"oidc.userGroup", o.UserGroup},
	} {
		if f.value == "" {
			return o, validation.Field(f.name, errors.New("required for single sign-on"))
		}
	}
	if o.AdminGroup == o.UserGroup {
		return o, validation.Field("oidc.userGroup", errors.New("must differ from the admin group"))
	}
	if err := sso.CheckURL(o.IssuerURL); err != nil {
		return o, validation.Field("oidc.issuerUrl", err)
	}
	if u, err := url.Parse(o.RedirectURL); err != nil || sso.CheckURL(o.RedirectURL) != nil ||
		u.Path != sso.CallbackPath {
		return o, validation.Field("oidc.redirectUrl", fmt.Errorf("must be https://<panel>%s", sso.CallbackPath))
	}
	if err := sso.Discover(ctx, o.IssuerURL); err != nil {
		return o, validation.Field("oidc.issuerUrl", err)
	}
	return o, nil
}

// OIDCClientSecret returns the client secret of the identity provider ("" when none is stored). Like Current,
// it is kept in memory until the settings change.
func (s *Store) OIDCClientSecret(ctx context.Context) (string, error) {
	return s.secret.get(ctx, func(ctx context.Context) (string, error) {
		secret := &corev1.Secret{}
		err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: OIDCSecret}, secret)
		if err != nil && !apierrors.IsNotFound(err) {
			return "", err
		}
		return string(secret.Data[oidcSecretKey]), nil
	})
}

// UpdateOIDCClientSecret stores the client secret of the identity provider ("" removes it).
func (s *Store) UpdateOIDCClientSecret(ctx context.Context, value string) error {
	secret := &corev1.Secret{}
	secret.Name, secret.Namespace = OIDCSecret, s.Namespace
	err := kube.CreateOrPatch(ctx, s.Reader, s.Client, secret, func() error {
		secret.Labels = map[string]string{tenancy.LabelManagedBy: tenancy.ManagedBy}
		secret.Data = map[string][]byte{oidcSecretKey: []byte(value)}
		return nil
	})
	s.Forget()
	return err
}
