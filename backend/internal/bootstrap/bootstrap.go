// Package bootstrap prepares the cluster when the panel starts: CRDs, the panel namespace,
// the session signing key, the first administrator and the cache configuration.
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
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"app/api/v1alpha1"
	"app/config/crd"
	"app/internal/auth"
	"app/internal/tenancy"
)

const fieldOwner = client.FieldOwner("kubedactyl")

// InstallCRDs applies the embedded CRDs with server-side apply and waits until they are established.
func InstallCRDs(ctx context.Context, c client.Client) error {
	names, err := applyCRDs(ctx, c)
	if err != nil {
		return err
	}
	return waitEstablished(ctx, c, names)
}

// applyCRDs server-side applies the embedded CRD manifests and returns their names.
func applyCRDs(ctx context.Context, c client.Client) ([]string, error) {
	entries, err := fs.ReadDir(crd.Files, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		data, err := fs.ReadFile(crd.Files, e.Name())
		if err != nil {
			return nil, err
		}
		u := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(data, &u.Object); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if err := c.Apply(
			ctx, client.ApplyConfigurationFromUnstructured(u), fieldOwner, client.ForceOwnership,
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

// EnsureNamespace creates the panel namespace if needed.
func EnsureNamespace(ctx context.Context, c client.Client, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   name,
		Labels: map[string]string{tenancy.LabelManagedBy: tenancy.ManagedBy},
	}}
	if err := c.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Only objects created by the panel are cached; everything else is read directly, so the
// panel never keeps all pods or secrets of the cluster in memory.
func CacheOptions() cache.Options {
	managed := labels.SelectorFromSet(labels.Set{tenancy.LabelManagedBy: tenancy.ManagedBy})
	return cache.Options{ByObject: map[client.Object]cache.ByObject{
		&corev1.Pod{}:                   {Label: managed},
		&corev1.Service{}:               {Label: managed},
		&corev1.PersistentVolumeClaim{}: {Label: managed},
		&corev1.ConfigMap{}:             {Label: managed},
		&networkingv1.NetworkPolicy{}:   {Label: managed},
	}}
}

func UncachedTypes() []client.Object {
	return []client.Object{
		&corev1.Secret{},
		&corev1.Namespace{},
		&corev1.PersistentVolume{},
		&apiextensionsv1.CustomResourceDefinition{},
		&rbacv1.RoleBinding{},
	}
}

const sessionKeySecret = "kubedactyl-auth"

// LoadSessionSigner reads (or creates) the random key that signs session tokens.
func LoadSessionSigner(ctx context.Context, c client.Client, ns string) (*auth.Signer, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: sessionKeySecret}, secret)
	if apierrors.IsNotFound(err) {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sessionKeySecret,
				Namespace: ns,
				Labels:    map[string]string{tenancy.LabelManagedBy: tenancy.ManagedBy},
			},
			Data: map[string][]byte{"session-key": key},
		}
		// Two panels starting at once: the one that loses the race uses the stored key.
		switch err := c.Create(ctx, secret); {
		case apierrors.IsAlreadyExists(err):
			return LoadSessionSigner(ctx, c, ns)
		case err != nil:
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return auth.NewSigner(secret.Data["session-key"])
}

// Admin creates the user "admin" from KUBEDACTYL_ADMIN_PASSWORD when no active
// administrator exists (for automated installations). Without the variable the web UI
// shows the setup page, which needs the setup token returned here (from
// KUBEDACTYL_SETUP_TOKEN or generated and logged once). The token is empty when an
// administrator exists.
func Admin(ctx context.Context, c client.Client, ns, url string, log *slog.Logger) (string, error) {
	var users v1alpha1.UserList
	if err := c.List(ctx, &users, client.InNamespace(ns)); err != nil {
		return "", err
	}
	for _, u := range users.Items {
		if u.Spec.Role == v1alpha1.RoleAdmin && !u.Spec.Disabled {
			return "", nil
		}
	}
	password := os.Getenv("KUBEDACTYL_ADMIN_PASSWORD")
	if password == "" {
		token := os.Getenv("KUBEDACTYL_SETUP_TOKEN")
		if token == "" {
			token = auth.NewSetupToken()
			log.Warn(
				"no administrator yet, open the setup page with this one-time link", "url", url+"/setup?token="+token,
			)
		} else {
			log.Warn("no administrator yet, open the setup page with KUBEDACTYL_SETUP_TOKEN", "url", url+"/setup")
		}
		return token, nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("KUBEDACTYL_ADMIN_PASSWORD: %w", err)
	}
	admin := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: "admin", Namespace: ns},
		Spec:       v1alpha1.UserSpec{DisplayName: "Administrator", Role: v1alpha1.RoleAdmin, PasswordHash: hash},
	}
	if err := c.Create(ctx, admin); err != nil {
		return "", err
	}
	log.Info("created initial admin user from KUBEDACTYL_ADMIN_PASSWORD", "user", "admin")
	return "", nil
}
