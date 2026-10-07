package sso

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"app/api/v1alpha1"
	"app/internal/testutil"
)

func config(idp *testutil.IdP) Config {
	return Config{
		OIDCSettings: v1alpha1.OIDCSettings{
			Enabled: true, IssuerURL: idp.URL, ClientID: idp.ClientID,
			UsernameClaim: DefaultUsernameClaim, GroupsClaim: DefaultGroupsClaim,
			AdminGroup: "admins", UserGroup: "players", RedirectURL: "https://panel.test" + CallbackPath,
		},
		ClientSecret: idp.Secret,
	}
}

// signIn starts a sign-in and follows the identity provider's redirect; it returns the flow and the callback.
func signIn(t *testing.T, c *Client, cfg Config) (Flow, url.Values) {
	t.Helper()
	authURL, flow, err := c.Start(context.Background(), cfg, "/servers/a")
	if err != nil {
		t.Fatal(err)
	}
	res, err := testutil.NoRedirects.Get(authURL)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	back, err := url.Parse(res.Header.Get("Location"))
	if err != nil || res.StatusCode != http.StatusFound {
		t.Fatalf("identity provider answered %d %v", res.StatusCode, err)
	}
	return flow, back.Query()
}

func TestSignIn(t *testing.T) {
	idp := testutil.NewIdP(t)
	var c Client
	cases := []struct {
		groups any
		role   v1alpha1.UserRole
		err    error
	}{
		{[]any{"players", "admins"}, v1alpha1.RoleAdmin, nil},
		{[]any{"players"}, v1alpha1.RoleUser, nil},
		{"admins", v1alpha1.RoleAdmin, nil},
		{[]any{"others"}, "", ErrNoAccess},
		{nil, "", ErrNoAccess},
	}
	for _, tc := range cases {
		idp.SignInAs(map[string]any{
			"sub": "u-1", "preferred_username": "alice", "name": "Alice A.", "email": "alice@example.com",
			"groups": tc.groups,
		})
		flow, back := signIn(t, &c, config(idp))
		acc, err := c.Finish(context.Background(), config(idp), flow, back.Get("state"), back.Get("code"))
		if !errors.Is(err, tc.err) {
			t.Fatalf("groups %v: err = %v, want %v", tc.groups, err, tc.err)
		}
		want := Account{
			Identity: v1alpha1.OIDCIdentity{Issuer: idp.URL, Subject: "u-1"},
			Username: "alice", DisplayName: "Alice A.", Email: "alice@example.com", Role: tc.role,
		}
		if err == nil && acc != want {
			t.Errorf("groups %v: account = %+v, want %+v", tc.groups, acc, want)
		}
	}
	if flow, _ := signIn(t, &c, config(idp)); flow.Next != "/servers/a" {
		t.Errorf("next = %q", flow.Next)
	}
}

func TestSignInRefusesForeignCallbacks(t *testing.T) {
	idp := testutil.NewIdP(t)
	var c Client
	idp.SignInAs(map[string]any{"sub": "u-1", "preferred_username": "alice", "groups": []any{"admins"}})
	ctx := context.Background()

	// A flow completes once: its cookie cannot be replayed.
	flow, back := signIn(t, &c, config(idp))
	if _, err := c.Finish(ctx, config(idp), flow, back.Get("state"), back.Get("code")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Finish(ctx, config(idp), flow, back.Get("state"), back.Get("code")); !errors.Is(err, ErrFlow) {
		t.Errorf("replayed flow: %v", err)
	}

	// Another browser's callback (login CSRF): the state does not match this browser's flow.
	flow, back = signIn(t, &c, config(idp))
	other, _ := signIn(t, &c, config(idp))
	if _, err := c.Finish(ctx, config(idp), other, back.Get("state"), back.Get("code")); !errors.Is(err, ErrFlow) {
		t.Errorf("foreign state: %v", err)
	}
	if _, err := c.Finish(ctx, config(idp), Flow{}, "", back.Get("code")); !errors.Is(err, ErrFlow) {
		t.Errorf("no flow: %v", err)
	}
	// The code belongs to another PKCE verifier.
	stolen := flow
	stolen.Verifier = other.Verifier
	if _, err := c.Finish(ctx, config(idp), stolen, back.Get("state"), back.Get("code")); err == nil {
		t.Error("a code was exchanged with another PKCE verifier")
	}
	// The ID token was issued for another sign-in (nonce).
	flow, back = signIn(t, &c, config(idp))
	flow.Nonce = "other"
	if _, err := c.Finish(ctx, config(idp), flow, back.Get("state"), back.Get("code")); !errors.Is(err, ErrFlow) {
		t.Errorf("foreign nonce: %v", err)
	}
	// Wrong client secret and an ID token for another client.
	cfg := config(idp)
	cfg.ClientSecret = "wrong"
	flow, back = signIn(t, &c, cfg)
	if _, err := c.Finish(ctx, cfg, flow, back.Get("state"), back.Get("code")); err == nil {
		t.Error("signed in with a wrong client secret")
	}
	cfg = config(idp)
	verifier := (&Client{}).verifierFor(t, cfg)
	if _, err := verifier(idp.IDToken(map[string]any{"sub": "u-1", "aud": "other-client"})); err == nil {
		t.Error("accepted an ID token of another client")
	}
}

// verifierFor verifies raw ID tokens like Finish does.
func (c *Client) verifierFor(t *testing.T, cfg Config) func(string) (any, error) {
	t.Helper()
	p, err := c.providerFor(context.Background(), cfg.IssuerURL)
	if err != nil {
		t.Fatal(err)
	}
	return func(raw string) (any, error) {
		return p.Verifier(oidcConfig(cfg)).Verify(context.Background(), raw)
	}
}

func TestCheckURL(t *testing.T) {
	for issuer, ok := range map[string]bool{
		"https://auth.example.com/realms/games": true,
		"http://localhost:8080/realms/games":    true,
		"http://127.0.0.1:8080":                 true,
		"http://auth.example.com/realms/games":  false,
		"https://user:pw@auth.example.com":      false,
		"https://auth.example.com/?x=1":         false,
		"auth.example.com":                      false,
		"":                                      false,
	} {
		if err := CheckURL(issuer); (err == nil) != ok {
			t.Errorf("CheckURL(%q) = %v", issuer, err)
		}
	}
}

// TestDiscoveryFailureIsKept: an identity provider that does not answer is not asked again for every sign-in.
func TestDiscoveryFailureIsKept(t *testing.T) {
	var c Client
	cfg := Config{OIDCSettings: v1alpha1.OIDCSettings{IssuerURL: "http://127.0.0.1:1"}}
	if _, _, err := c.Start(context.Background(), cfg, "/"); err == nil {
		t.Fatal("started a sign-in without an identity provider")
	}
	failed := c.failed
	if _, _, err := c.Start(context.Background(), cfg, "/"); err == nil || c.failed != failed {
		t.Errorf("asked again at once: %v", err)
	}
}
