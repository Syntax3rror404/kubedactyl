package tenancy

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureNamespaceEnforcesPodSecurity(t *testing.T) {
	NamespacePrefix = "kubedactyl-user-"
	old := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kubedactyl-user-bob", Labels: map[string]string{LabelUser: "bob"}},
	}
	foreign := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kubedactyl-user-eve"}}
	c := fake.NewClientBuilder().WithObjects(old, foreign).Build()
	ctx := context.Background()
	for _, user := range []string{"alice", "bob"} { // new, and created by an older version
		name, err := EnsureNamespace(ctx, c, user)
		if err != nil {
			t.Fatal(err)
		}
		ns := &corev1.Namespace{}
		_ = c.Get(ctx, client.ObjectKey{Name: name}, ns)
		if ns.Labels["pod-security.kubernetes.io/enforce"] != "baseline" || ns.Labels[LabelUser] != user {
			t.Errorf("%s: labels %v", name, ns.Labels)
		}
	}
	if _, err := EnsureNamespace(ctx, c, "eve"); err == nil {
		t.Error("a namespace of the same name that is not the user's must not be taken over")
	}
}

func TestEnsureNamespaceGrantsTheTenantRole(t *testing.T) {
	SystemNamespace, NamespacePrefix = "kubedactyl", "kubedactyl-user-"
	TenantRole, PanelServiceAccount = "kubedactyl-tenant", "kubedactyl"
	t.Cleanup(func() { TenantRole, PanelServiceAccount = "", "" })
	wrong := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: TenantBinding, Namespace: "kubedactyl-user-bob"},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "something-else"},
	}
	c := fake.NewClientBuilder().WithObjects(wrong).Build()
	ctx := context.Background()
	for _, user := range []string{"alice", "bob"} {
		name, err := EnsureNamespace(ctx, c, user)
		if err != nil {
			t.Fatal(err)
		}
		rb := &rbacv1.RoleBinding{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: name, Name: TenantBinding}, rb); err != nil {
			t.Fatal(err)
		}
		if rb.RoleRef.Name != "kubedactyl-tenant" || len(rb.Subjects) != 1 ||
			rb.Subjects[0].Name != "kubedactyl" || rb.Subjects[0].Namespace != "kubedactyl" {
			t.Errorf("%s: binding %+v %+v", name, rb.RoleRef, rb.Subjects)
		}
	}
}
