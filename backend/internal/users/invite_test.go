package users

import (
	"context"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"app/api/v1alpha1"
	"app/internal/tenancy"
	"app/internal/testutil"
)

// TestResetLinksOfAccountsThatCannotUseThem: a reset link stops working once its account is linked to the
// identity provider (linking deletes it) or disabled.
func TestResetLinksOfAccountsThatCannotUseThem(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		label  string
		change func(s *Store, u *v1alpha1.User) error
	}{
		{"linked to the identity provider", func(s *Store, u *v1alpha1.User) error {
			_, err := s.SignInOIDC(ctx, account("s-1", u.Name, v1alpha1.RoleUser), opts(true, true))
			return err
		}},
		{"linked meanwhile", func(s *Store, u *v1alpha1.User) error {
			u.Spec.OIDC = &v1alpha1.OIDCIdentity{Issuer: "https://idp", Subject: "s-1"}
			return s.Client.Update(ctx, u)
		}},
		{"disabled", func(s *Store, u *v1alpha1.User) error {
			u.Spec.Disabled = true
			return s.Client.Update(ctx, u)
		}},
	}
	for _, tc := range cases {
		s := newStore(t)
		u, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"})
		if err != nil {
			t.Fatal(err)
		}
		_, token, err := s.CreateInvite(ctx, InviteInput{Reset: true, Username: "alice"}, "admin")
		if err != nil {
			t.Fatal(err)
		}
		if err := tc.change(s, u); err != nil {
			t.Fatalf("%s: %v", tc.label, err)
		}
		_, _, err = s.AcceptInvite(ctx, token, AcceptInput{Password: "a-new-password"})
		if !errors.Is(err, ErrInvalidInvite) {
			t.Errorf("%s: %v, want an invalid link", tc.label, err)
		}
	}
}

func TestSignInOIDCDeletesResetLinks(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if _, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CreateInvite(ctx, InviteInput{Reset: true, Username: "alice"}, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CreateInvite(ctx, InviteInput{Username: "erin"}, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SignInOIDC(ctx, account("s-1", "alice", v1alpha1.RoleUser), opts(true, true)); err != nil {
		t.Fatal(err)
	}
	invites, _ := s.ListInvites(ctx, "")
	if len(invites) != 1 || invites[0].Spec.Username != "erin" {
		t.Errorf("only the reset link of the linked account is deleted: %+v", invites)
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
				return apierrors.NewConflict(UserResource, obj.GetName(), errors.New("changed"))
			}
			return c.Patch(ctx, obj, patch, opts...)
		},
	}).Build()
	s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	if _, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"}); err != nil {
		t.Fatal(err)
	}
	invite, token, err := s.CreateInvite(ctx, InviteInput{
		Owner: "alice", Limit: 10, Shares: []v1alpha1.InviteShare{{Server: "srv"}},
	}, "alice")
	if err != nil {
		t.Fatal(err)
	}
	u, used, err := s.AcceptInvite(ctx, token, AcceptInput{Username: "carol", Password: "a-long-password"})
	if err != nil || u.Name != "carol" || used.Name != invite.Name {
		t.Fatalf("accept: %v %v %v", u, used, err)
	}
	_ = c.Get(ctx, client.ObjectKeyFromObject(gs), gs)
	if len(gs.Spec.SharedWith) != 1 || gs.Spec.SharedWith[0].User != "carol" {
		t.Errorf("shares: %+v", gs.Spec.SharedWith)
	}
	if _, err := s.FindInvite(ctx, token); !errors.Is(err, ErrInvalidInvite) {
		t.Errorf("the link still works: %v", err)
	}
}
