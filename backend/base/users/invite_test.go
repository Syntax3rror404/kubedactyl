package users

import (
	"context"
	"errors"
	"testing"

	"app/api/v1alpha1"
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
			_, err := s.SignInOIDC(ctx, account("s-1", u.Name, v1alpha1.DefaultRole), opts(true, true))
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
		_, token, err := s.CreateInvite(ctx, InviteInput{Reset: true, Username: "alice"}, admin)
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
	if _, _, err := s.CreateInvite(ctx, InviteInput{Reset: true, Username: "alice"}, admin); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CreateInvite(ctx, InviteInput{Username: "erin"}, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SignInOIDC(ctx, account("s-1", "alice", v1alpha1.DefaultRole), opts(true, true)); err != nil {
		t.Fatal(err)
	}
	invites, _ := s.ListInvites(ctx, admin)
	if len(invites) != 1 || invites[0].Spec.Username != "erin" {
		t.Errorf("only the reset link of the linked account is deleted: %+v", invites)
	}
}
