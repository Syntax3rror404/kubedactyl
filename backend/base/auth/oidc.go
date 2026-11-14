package auth

// Single sign-on through an OpenID Connect identity provider: the authorization code flow with PKCE, state and
// nonce, and the account the verified ID token describes (username, display name, email and the role its groups
// grant).

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"app/api/v1alpha1"
)

// Claims read when the settings name none.
const (
	DefaultUsernameClaim = "preferred_username"
	DefaultGroupsClaim   = "groups"
)

// Errors of a sign-in.
var (
	// ErrNoAccess means the user is in neither the admin nor the member group.
	ErrNoAccess = errors.New("the user is in neither the admin nor the " + string(v1alpha1.DefaultRole) + " group")
	// ErrOIDCFlow means the callback does not belong to the sign-in this browser started (state or nonce).
	ErrOIDCFlow = errors.New("the sign-in does not match the one this browser started")
)

// oidcHTTPClient talks to the identity provider; a provider that does not answer must not hold a sign-in forever.
var oidcHTTPClient = &http.Client{Timeout: 10 * time.Second}

// OIDCFlowLifetime is how long a started sign-in can be completed.
const OIDCFlowLifetime = 10 * time.Minute

// oidcFailureTTL is how long a failed discovery is remembered: an identity provider that does not answer is not
// asked again for every sign-in.
const oidcFailureTTL = 30 * time.Second

// maxUsedOIDCFlows bounds the memory of completed sign-ins; when it is full, the one that ends first is dropped (its
// code is spent at the identity provider anyway).
const maxUsedOIDCFlows = 10000

// OIDCConfig is the client configuration of a sign-in.
type OIDCConfig struct {
	v1alpha1.OIDCSettings
	// ClientSecret is empty for a public client (PKCE only).
	ClientSecret string
}

// OIDCFlow is what the browser keeps between the redirect to the identity provider and the callback: state ties
// the callback to this browser, nonce the ID token to this sign-in and the PKCE verifier the code to this client.
type OIDCFlow struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	// Next is the page of the panel to open after the sign-in.
	Next string `json:"r"`
}

// OIDCAccount is the user an ID token describes.
type OIDCAccount struct {
	Identity    v1alpha1.OIDCIdentity
	Username    string
	DisplayName string
	Email       string
	Role        v1alpha1.UserRole
}

// OIDC keeps the discovery document and signing keys of the issuer in memory (the keys are fetched again
// when the identity provider rotates them) and the states of completed sign-ins until they expire, so a flow
// is completed once.
type OIDC struct {
	mu       sync.Mutex
	issuer   string
	provider *oidc.Provider
	failed   time.Time // last failed discovery of issuer
	err      error

	usedMu    sync.Mutex
	usedFlows map[string]time.Time // state → end of its flow, guarded by usedMu
}

// OIDCCallbackPath is the callback of the panel the identity provider redirects to.
const OIDCCallbackPath = "/api/auth/oidc/callback"

// CheckOIDCURL accepts an https URL, or http for a loopback address (a provider or panel on the same machine).
func CheckOIDCURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("not a valid URL")
	}
	if ip := net.ParseIP(u.Hostname()); u.Scheme == "http" && (u.Hostname() == "localhost" || ip.IsLoopback()) {
		return nil
	}
	if u.Scheme != "https" {
		return errors.New("must start with https://")
	}
	return nil
}

// DiscoverOIDC reads the discovery document of an issuer (not kept): for checks of the settings and the health.
func DiscoverOIDC(ctx context.Context, issuer string) error {
	_, err := discover(ctx, issuer)
	return err
}

func discover(ctx context.Context, issuer string) (*oidc.Provider, error) {
	p, err := oidc.NewProvider(oidc.ClientContext(ctx, oidcHTTPClient), issuer)
	if err != nil {
		return nil, fmt.Errorf("discovery of %s failed: %w", issuer, err)
	}
	return p, nil
}

// providerFor returns the provider of the issuer, discovered once; a failure is returned again for failureTTL.
func (c *OIDC) providerFor(ctx context.Context, issuer string) (*oidc.Provider, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.issuer == issuer && c.provider != nil {
		return c.provider, nil
	}
	if c.issuer == issuer && time.Since(c.failed) < oidcFailureTTL {
		return nil, c.err
	}
	p, err := discover(ctx, issuer)
	c.issuer, c.provider, c.err = issuer, p, err
	if err != nil {
		c.failed = time.Now()
	}
	return p, err
}

// used reports whether the flow of state was completed before.
func (c *OIDC) used(state string) bool {
	c.usedMu.Lock()
	defer c.usedMu.Unlock()
	_, ok := c.usedFlows[state]
	return ok
}

// complete marks the state of a verified flow as used; false when it was used before.
func (c *OIDC) complete(state string, now time.Time) bool {
	c.usedMu.Lock()
	defer c.usedMu.Unlock()
	if c.usedFlows == nil {
		c.usedFlows = map[string]time.Time{}
	}
	if _, ok := c.usedFlows[state]; ok {
		return false
	}
	for s, end := range c.usedFlows {
		if now.After(end) {
			delete(c.usedFlows, s)
		}
	}
	if len(c.usedFlows) >= maxUsedOIDCFlows {
		first := ""
		for s, end := range c.usedFlows {
			if first == "" || end.Before(c.usedFlows[first]) {
				first = s
			}
		}
		delete(c.usedFlows, first)
	}
	c.usedFlows[state] = now.Add(OIDCFlowLifetime)
	return true
}

func verifierConfig(cfg OIDCConfig) *oidc.Config { return &oidc.Config{ClientID: cfg.ClientID} }

func oauthConfig(p *oidc.Provider, cfg OIDCConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.OIDCSettings.RedirectURL,
		Endpoint: p.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

// Start begins a sign-in: it returns the URL of the identity provider's sign-in page and the flow the browser
// keeps until the callback.
func (c *OIDC) Start(ctx context.Context, cfg OIDCConfig, next string) (string, OIDCFlow, error) {
	p, err := c.providerFor(ctx, cfg.IssuerURL)
	if err != nil {
		return "", OIDCFlow{}, err
	}
	flow := OIDCFlow{State: rand.Text(), Nonce: rand.Text(), Verifier: oauth2.GenerateVerifier(), Next: next}
	authURL := oauthConfig(p, cfg).AuthCodeURL(
		flow.State, oidc.Nonce(flow.Nonce), oauth2.S256ChallengeOption(flow.Verifier),
	)
	return authURL, flow, nil
}

// Finish completes a sign-in with the state and code of the callback (once per flow): it exchanges the code for
// the tokens, verifies the ID token (signature, issuer, audience, expiry, nonce) and returns the account it
// describes. Only a verified flow is marked as used, so failed callbacks cannot fill that memory.
func (c *OIDC) Finish(ctx context.Context, cfg OIDCConfig, flow OIDCFlow, state, code string) (OIDCAccount, error) {
	if flow.State == "" || subtle.ConstantTimeCompare([]byte(state), []byte(flow.State)) != 1 || c.used(flow.State) {
		return OIDCAccount{}, ErrOIDCFlow
	}
	p, err := c.providerFor(ctx, cfg.IssuerURL)
	if err != nil {
		return OIDCAccount{}, err
	}
	tok, err := oauthConfig(p, cfg).Exchange(
		oidc.ClientContext(ctx, oidcHTTPClient), code, oauth2.VerifierOption(flow.Verifier),
	)
	if err != nil {
		return OIDCAccount{}, fmt.Errorf("exchanging the code: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return OIDCAccount{}, errors.New("the identity provider sent no ID token")
	}
	idt, err := p.Verifier(verifierConfig(cfg)).Verify(ctx, raw)
	if err != nil {
		return OIDCAccount{}, fmt.Errorf("verifying the ID token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idt.Nonce), []byte(flow.Nonce)) != 1 || !c.complete(flow.State, time.Now()) {
		return OIDCAccount{}, ErrOIDCFlow
	}
	var claims map[string]any
	if err := idt.Claims(&claims); err != nil {
		return OIDCAccount{}, err
	}
	return oidcAccountOf(cfg.OIDCSettings, v1alpha1.OIDCIdentity{Issuer: idt.Issuer, Subject: idt.Subject}, claims)
}

// oidcAccountOf reads the account from the claims of an ID token; the groups decide the role. Without access the
// account is returned with the error, for the log.
func oidcAccountOf(s v1alpha1.OIDCSettings, id v1alpha1.OIDCIdentity, claims map[string]any) (OIDCAccount, error) {
	acc := OIDCAccount{Identity: id}
	// Taken as it is: lower-casing would make "Admin" and "admin" of the identity provider the same account.
	acc.Username, _ = claims[s.UsernameClaim].(string)
	acc.DisplayName, _ = claims["name"].(string)
	acc.Email, _ = claims["email"].(string)
	groups := claimStrings(claims[s.GroupsClaim])
	switch {
	case slices.Contains(groups, s.AdminGroup):
		acc.Role = v1alpha1.RoleAdmin
	case slices.Contains(groups, s.MemberGroup):
		acc.Role = v1alpha1.DefaultRole
	default:
		return acc, ErrNoAccess
	}
	return acc, nil
}

// claimStrings reads a claim that is a list of strings or a single string.
func claimStrings(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
