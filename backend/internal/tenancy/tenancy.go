// Package tenancy maps panel users to their Kubernetes namespaces.
package tenancy

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Labels on user namespaces and game servers.
const (
	LabelUser      = "kubedactyl.io/user"
	LabelOwner     = "kubedactyl.io/owner"
	LabelManagedBy = "app.kubernetes.io/managed-by"
	ManagedBy      = "kubedactyl"
)

// SystemNamespace is the panel namespace (eggs, users, settings, session key, hardware probe and
// upgrade pods). Game servers live in the user namespaces.
var SystemNamespace = "kubedactyl"

// Owns reports whether a namespace belongs to this installation. Several installations may
// share a cluster; each one only touches its own namespaces.
func Owns(namespace string) bool {
	return namespace == SystemNamespace || strings.HasPrefix(namespace, NamespacePrefix)
}

// NamespacePrefix is prepended to the username to form the user namespace. main sets it
// to "<panel namespace>-user-", so several installations in one cluster do not collide.
var NamespacePrefix = "kubedactyl-user-"

var usernameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)

// ValidateUsername enforces names that are valid object and namespace names.
func ValidateUsername(name string) error {
	if !usernameRe.MatchString(name) {
		return errors.New(
			"username must be 3-32 characters: lowercase letters, digits and dashes, starting with a letter",
		)
	}
	return nil
}

// TenantRole is the cluster role with the panel's permissions inside user namespaces and
// PanelServiceAccount the panel's service account (in SystemNamespace). main sets both when
// the panel runs in the cluster; the panel then binds the role in every user namespace. Empty
// (e.g. a local panel with an administrator's kubeconfig): no binding is needed.
var TenantRole, PanelServiceAccount string

// TenantBinding is the name of the role binding in every user namespace.
const TenantBinding = "kubedactyl-panel"

// Namespace returns the namespace of a user.
func Namespace(user string) string { return NamespacePrefix + user }

// namespaceLabels mark a user namespace and enforce the Pod Security "baseline" level: no
// privileged pods, host namespaces, host paths or extra capabilities beyond the default set,
// whatever creates the pod.
func namespaceLabels(user string) map[string]string {
	return map[string]string{
		LabelManagedBy:                       ManagedBy,
		LabelUser:                            user,
		"pod-security.kubernetes.io/enforce": "baseline",
		"pod-security.kubernetes.io/enforce-version": "latest",
		"pod-security.kubernetes.io/warn":            "baseline",
	}
}

// EnsureNamespace creates the namespace of a user (or adds missing labels to it) and grants
// the panel its tenant role there.
func EnsureNamespace(ctx context.Context, c client.Client, user string) (string, error) {
	name, err := ensureNamespaceObject(ctx, c, user)
	if err != nil {
		return "", err
	}
	return name, ensureTenantBinding(ctx, c, name)
}

func ensureNamespaceObject(ctx context.Context, c client.Client, user string) (string, error) {
	name := Namespace(user)
	want := namespaceLabels(user)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: want}}
	err := c.Create(ctx, ns)
	if !apierrors.IsAlreadyExists(err) {
		return name, err
	}
	if err := c.Get(ctx, client.ObjectKey{Name: name}, ns); err != nil {
		return "", err
	}
	if ns.Labels[LabelUser] != user {
		return "", fmt.Errorf("namespace %s exists but does not belong to user %s", name, user)
	}
	patch := client.MergeFrom(ns.DeepCopy())
	changed := false
	for k, v := range want {
		if ns.Labels[k] != v {
			ns.Labels[k], changed = v, true
		}
	}
	if !changed {
		return name, nil
	}
	return name, c.Patch(ctx, ns, patch)
}

// ensureTenantBinding binds the tenant role to the panel in a user namespace. The role
// reference of a binding cannot change, so a binding that differs is replaced.
func ensureTenantBinding(ctx context.Context, c client.Client, namespace string) error {
	if TenantRole == "" || PanelServiceAccount == "" {
		return nil
	}
	want := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TenantBinding,
			Namespace: namespace,
			Labels:    map[string]string{LabelManagedBy: ManagedBy},
		},
		RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: TenantRole},
		Subjects: []rbacv1.Subject{
			{Kind: rbacv1.ServiceAccountKind, Name: PanelServiceAccount, Namespace: SystemNamespace},
		},
	}
	err := c.Create(ctx, want)
	if !apierrors.IsAlreadyExists(err) {
		return err
	}
	cur := &rbacv1.RoleBinding{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(want), cur); err != nil {
		return err
	}
	if cur.RoleRef == want.RoleRef && slices.Equal(cur.Subjects, want.Subjects) {
		return nil
	}
	if err := c.Delete(ctx, cur); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return c.Create(ctx, want)
}
