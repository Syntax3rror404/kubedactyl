package appusers

import (
	"context"
	"errors"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"app/api/v1alpha1"
	"app/app"
	"app/base/settings"
	"app/base/users"
	"app/internal/appsettings"
	"app/internal/tenancy"
	"app/internal/testutil"
)

// newStore is the account store of the panel: base accounts with the subusers of Kubedactyl.
func newStore(c client.Client) *users.Store {
	set := &settings.Store{Client: c, Reader: c, Namespace: testutil.Namespace, App: appsettings.Settings{Reader: c}}
	return &users.Store{Client: c, Reader: c, Namespace: testutil.Namespace, App: &Accounts{
		Client: c, Reader: c, Namespace: testutil.Namespace, Settings: set,
	}}
}

func TestCheckActive(t *testing.T) {
	ctx := context.Background()
	deleting := &metav1.Time{Time: time.Now()}
	user := func(name, owner string, disabled bool, deleted *metav1.Time) *v1alpha1.User {
		return &v1alpha1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: testutil.Namespace, DeletionTimestamp: deleted, Finalizers: []string{"test"},
			},
			Spec: v1alpha1.UserSpec{Owner: owner, Disabled: disabled},
		}
	}
	c := testutil.Builder(t).WithObjects(
		user("alice", "", false, nil), user("bob", "", true, nil), user("dan", "", false, deleting),
	).Build()
	s := newStore(c)
	cases := []struct {
		label string
		user  *v1alpha1.User
		want  error
	}{
		{"enabled", user("alice", "", false, nil), nil},
		{"disabled", user("bob", "", true, nil), users.ErrDisabled},
		{"being deleted", user("dan", "", false, deleting), users.ErrDisabled},
		{"subuser", user("carol", "alice", false, nil), nil},
		{"subuser of a disabled owner", user("carol", "bob", false, nil), users.ErrDisabled},
		{"subuser of an owner being deleted", user("carol", "dan", false, nil), users.ErrDisabled},
		{"subuser of an owner that is gone", user("carol", "gone", false, nil), users.ErrDisabled},
	}
	for _, tc := range cases {
		if err := s.CheckActive(ctx, tc.user); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", tc.label, err, tc.want)
		}
	}
}

// TestAcceptInviteSharesDespiteConflicts: a server changed while the invite shares it is read again, and the
// link is spent once the account exists.
func TestAcceptInviteSharesDespiteConflicts(t *testing.T) {
	ctx := context.Background()
	conflicts := 2
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "srv", Namespace: tenancy.Namespace("alice")}}
	c := testutil.Builder(t).WithObjects(gs).WithInterceptorFuncs(interceptor.Funcs{
		Patch: func(
			ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption,
		) error {
			if _, ok := obj.(*v1alpha1.GameServer); ok && conflicts > 0 {
				conflicts--
				return apierrors.NewConflict(users.UserResource, obj.GetName(), errors.New("changed"))
			}
			return c.Patch(ctx, obj, patch, opts...)
		},
	}).Build()
	s := newStore(c)
	alice, err := s.Create(ctx, users.CreateInput{Username: "alice", Password: "a-long-password"})
	if err != nil {
		t.Fatal(err)
	}
	invite, token, err := s.CreateInvite(ctx, users.InviteInput{
		InviteRequest: app.InviteRequest{Shares: []v1alpha1.InviteShare{{Server: "srv"}}},
	}, alice)
	if err != nil {
		t.Fatal(err)
	}
	u, used, err := s.AcceptInvite(ctx, token, users.AcceptInput{Username: "carol", Password: "a-long-password"})
	if err != nil || u.Name != "carol" || used.Name != invite.Name {
		t.Fatalf("accept: %v %v %v", u, used, err)
	}
	_ = c.Get(ctx, client.ObjectKeyFromObject(gs), gs)
	if len(gs.Spec.SharedWith) != 1 || gs.Spec.SharedWith[0].User != "carol" {
		t.Errorf("shares: %+v", gs.Spec.SharedWith)
	}
	if _, err := s.FindInvite(ctx, token); !errors.Is(err, users.ErrInvalidInvite) {
		t.Errorf("the link still works: %v", err)
	}
}
