package bootstrap

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/testutil"
)

func TestAdmin(t *testing.T) {
	user := func(name string, spec v1alpha1.UserSpec) client.Object {
		return &v1alpha1.User{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testutil.Namespace}, Spec: spec}
	}
	active := v1alpha1.UserSpec{Role: v1alpha1.RoleAdmin}
	disabled := v1alpha1.UserSpec{Role: v1alpha1.RoleAdmin, Disabled: true}
	demoted := v1alpha1.UserSpec{Role: v1alpha1.RoleUser}
	cases := []struct {
		name      string
		existing  []client.Object
		password  string
		wantToken bool
		wantRole  v1alpha1.UserRole // of "admin" afterwards ("" = none)
	}{
		{"administrator exists", []client.Object{user("alice", active)}, "secret-pw-123", false, ""},
		{"setup page", nil, "", true, ""},
		{"created from the environment", nil, "secret-pw-123", false, v1alpha1.RoleAdmin},
		{"disabled admin kept", []client.Object{user("admin", disabled)}, "secret-pw-123", true, v1alpha1.RoleAdmin},
		{"demoted admin kept", []client.Object{user("admin", demoted)}, "secret-pw-123", true, v1alpha1.RoleUser},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("KUBEDACTYL_ADMIN_PASSWORD", c.password)
			t.Setenv("KUBEDACTYL_SETUP_TOKEN", "")
			cl := testutil.Builder(t).WithObjects(c.existing...).Build()
			token, err := Admin(t.Context(), cl, testutil.Namespace, "http://localhost:8080", testutil.Logger())
			if err != nil || (token != "") != c.wantToken {
				t.Fatalf("token %q, err %v, want a token: %v", token, err, c.wantToken)
			}
			var admin v1alpha1.User
			_ = cl.Get(t.Context(), client.ObjectKey{Namespace: testutil.Namespace, Name: "admin"}, &admin)
			if admin.Spec.Role != c.wantRole {
				t.Errorf("admin role %q, want %q", admin.Spec.Role, c.wantRole)
			}
		})
	}
}
