package users

import (
	"context"
	"errors"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/sso"
)

// linking settings: keepPasswords and linkByUsername.
func opts(keep, link bool) v1alpha1.OIDCSettings {
	return v1alpha1.OIDCSettings{KeepPasswords: keep, LinkByUsername: link}
}

func account(sub, username string, role v1alpha1.UserRole) sso.Account {
	return sso.Account{
		Identity: v1alpha1.OIDCIdentity{Issuer: "https://idp", Subject: sub},
		Username: username, DisplayName: "Name " + sub, Email: sub + "@example.com", Role: role,
	}
}

func get(t *testing.T, s *Store, name string) *v1alpha1.User {
	t.Helper()
	u := &v1alpha1.User{}
	if err := s.Reader.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, u); err != nil {
		t.Fatal(err)
	}
	return u
}

func createAdmin(t *testing.T, s *Store, name string) {
	t.Helper()
	in := CreateInput{Username: name, Password: "a-long-password", Role: v1alpha1.RoleAdmin}
	if _, err := s.Create(context.Background(), in); err != nil {
		t.Fatal(err)
	}
}

func TestSignInOIDCCreatesAccounts(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, err := s.SignInOIDC(ctx, account("s-1", "carol", v1alpha1.RoleUser), opts(false, true))
	if err != nil {
		t.Fatal(err)
	}
	u = get(t, s, u.Name)
	if u.Spec.OIDC == nil || u.Spec.OIDC.Subject != "s-1" || u.Spec.PasswordHash != "" ||
		u.Spec.Email != "s-1@example.com" || u.Spec.Role != v1alpha1.RoleUser {
		t.Errorf("new account = %+v", u.Spec)
	}
	// Changes at the identity provider reach the account; a new username keeps the linked account.
	again, err := s.SignInOIDC(ctx, account("s-1", "carol-renamed", v1alpha1.RoleAdmin), opts(false, true))
	if err != nil || again.Name != "carol" || get(t, s, "carol").Spec.Role != v1alpha1.RoleAdmin {
		t.Errorf("second sign-in: %v %v", again, err)
	}
	_, err = s.SignInOIDC(ctx, account("s-2", "Not Valid", v1alpha1.RoleUser), opts(false, true))
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("invalid username: %v", err)
	}
}

func TestSignInOIDCLinksExistingAccounts(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	for _, name := range []string{"alice", "bob"} {
		createAdmin(t, s, name)
	}
	alice := get(t, s, "alice")
	alice.Spec.Tokens = []v1alpha1.APIToken{{ID: "t1"}}
	if err := s.Client.Update(ctx, alice); err != nil {
		t.Fatal(err)
	}
	// Without linkByUsername nobody gets an existing account by choosing its name at the identity provider.
	_, err := s.SignInOIDC(ctx, account("s-a", "alice", v1alpha1.RoleAdmin), opts(true, false))
	if !errors.Is(err, ErrNotLinked) || get(t, s, "alice").Spec.OIDC != nil {
		t.Errorf("linked without linkByUsername: %v", err)
	}
	_, err = s.SignInOIDC(ctx, account("s-a", "Alice", v1alpha1.RoleAdmin), opts(true, true))
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("another case of the name: %v", err)
	}
	// keepPassword: the local password keeps working next to single sign-on; sessions and API tokens made
	// before the link end.
	if _, err := s.SignInOIDC(ctx, account("s-a", "alice", v1alpha1.RoleAdmin), opts(true, true)); err != nil {
		t.Fatal(err)
	}
	u := get(t, s, "alice")
	if u.Spec.OIDC == nil || !auth.VerifyPassword("a-long-password", u.Spec.PasswordHash) {
		t.Errorf("alice = %+v, want linked with her password", u.Spec)
	}
	if u.Spec.SessionEpoch != alice.Spec.SessionEpoch+1 || len(u.Spec.Tokens) != 0 {
		t.Errorf("sessions and tokens of alice survived the link: %+v", u.Spec)
	}
	// Without: the account signs in only through the identity provider from now on.
	if _, err := s.SignInOIDC(ctx, account("s-b", "bob", v1alpha1.RoleAdmin), opts(false, true)); err != nil {
		t.Fatal(err)
	}
	bob := get(t, s, "bob")
	if bob.Spec.PasswordHash != "" {
		t.Error("bob kept his password")
	}
	// Another user of the identity provider with the same username cannot take the account over.
	_, err = s.SignInOIDC(ctx, account("s-x", "bob", v1alpha1.RoleAdmin), opts(false, true))
	if !errors.Is(err, ErrOtherIdentity) {
		t.Errorf("other identity: %v", err)
	}
	// The identity provider sets profile and role; an account without password gets none.
	name, role := "Bobby", v1alpha1.RoleUser
	if err := s.Update(ctx, bob, UpdateInput{DisplayName: &name}); !errors.Is(err, ErrManaged) {
		t.Errorf("display name of a linked account: %v", err)
	}
	if err := s.Update(ctx, bob, UpdateInput{Role: &role}); !errors.Is(err, ErrManaged) {
		t.Errorf("role of a linked account: %v", err)
	}
	same, pw := bob.Spec.DisplayName, "another-long-password"
	if err := s.Update(ctx, bob, UpdateInput{DisplayName: &same}); err != nil {
		t.Errorf("unchanged display name: %v", err)
	}
	if err := s.Update(ctx, bob, UpdateInput{Password: &pw}); !errors.Is(err, ErrNoPassword) {
		t.Errorf("password for an account without one: %v", err)
	}
	if err := s.UpdatePassword(ctx, bob, "", pw, false); !errors.Is(err, ErrNoPassword) {
		t.Errorf("own password without one: %v", err)
	}
}

func TestSignInOIDCKeepsLastAdminAndDisabled(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	createAdmin(t, s, "admin")
	// The identity provider removed the last administrator from the admin group: refused, not demoted.
	_, err := s.SignInOIDC(ctx, account("s-1", "admin", v1alpha1.RoleUser), opts(false, true))
	if !errors.Is(err, ErrLastAdmin) {
		t.Errorf("demoting the last admin: %v", err)
	}
	if u := get(t, s, "admin"); u.Spec.Role != v1alpha1.RoleAdmin || u.Spec.OIDC != nil {
		t.Errorf("admin changed: %+v", u.Spec)
	}
	user, err := s.SignInOIDC(ctx, account("s-2", "dave", v1alpha1.RoleUser), opts(false, true))
	if err != nil {
		t.Fatal(err)
	}
	disabled := true
	if err := s.Update(ctx, user, UpdateInput{Disabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	_, err = s.SignInOIDC(ctx, account("s-2", "dave", v1alpha1.RoleUser), opts(false, true))
	if !errors.Is(err, ErrDisabled) {
		t.Errorf("disabled account: %v", err)
	}
}
