// Package sso signs users in through an OpenID Connect identity provider: the authorization code flow with
// PKCE, state and nonce, and the account the verified ID token describes (username, display name, email and
// the role its groups grant).
package sso

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
	// ErrNoAccess: the user is in neither the admin nor the user group.
	ErrNoAccess = errors.New("the user is in neither the admin nor the user group")
	// ErrFlow: the callback does not belong to the sign-in this browser started (state or nonce).
	ErrFlow = errors.New("the sign-in does not match the one this browser started")
)

// httpClient talks to the identity provider; a provider that does not answer must not hold a sign-in forever.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// FlowLifetime is how long a started sign-in can be completed.
const FlowLifetime = 10 * time.Minute

// failureTTL is how long a failed discovery is remembered: an identity provider that does not answer is not
// asked again for every sign-in.
const failureTTL = 30 * time.Second

// maxUsedFlows bounds the memory of completed sign-ins; when it is full, further callbacks are refused until
// entries expire.
const maxUsedFlows = 10000

// Config is the client configuration of a sign-in.
type Config struct {
	v1alpha1.OIDCSettings
	// ClientSecret is empty for a public client (PKCE only).
	ClientSecret string
}

// Flow is what the browser keeps between the redirect to the identity provider and the callback: state ties
// the callback to this browser, nonce the ID token to this sign-in and the PKCE verifier the code to this client.
type Flow struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	// Next is the page of the panel to open after the sign-in.
	Next string `json:"r"`
}

// Account is the user an ID token describes.
type Account struct {
	Identity    v1alpha1.OIDCIdentity
	Username    string
	DisplayName string
	Email       string
	Role        v1alpha1.UserRole
}

// Client keeps the discovery document and signing keys of the issuer in memory (the keys are fetched again
// when the identity provider rotates them) and the states of completed sign-ins until they expire, so a flow
// is completed once.
type Client struct {
	mu       sync.Mutex
	issuer   string
	provider *oidc.Provider
	failed   time.Time // last failed discovery of issuer
	err      error

	usedMu sync.Mutex
	used   map[string]time.Time // state → end of its flow
}

// CallbackPath is the callback of the panel the identity provider redirects to.
const CallbackPath = "/api/auth/oidc/callback"

// CheckURL accepts an https URL, or http for a loopback address (a provider or panel on the same machine).
func CheckURL(raw string) error {
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

// Discover reads the discovery document of an issuer (not kept): for checks of the settings and the health.
func Discover(ctx context.Context, issuer string) error {
	_, err := discover(ctx, issuer)
	return err
}

func discover(ctx context.Context, issuer string) (*oidc.Provider, error) {
	p, err := oidc.NewProvider(oidc.ClientContext(ctx, httpClient), issuer)
	if err != nil {
		return nil, fmt.Errorf("discovery of %s failed: %w", issuer, err)
	}
	return p, nil
}

// providerFor returns the provider of the issuer, discovered once; a failure is returned again for failureTTL.
func (c *Client) providerFor(ctx context.Context, issuer string) (*oidc.Provider, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.issuer == issuer && c.provider != nil {
		return c.provider, nil
	}
	if c.issuer == issuer && time.Since(c.failed) < failureTTL {
		return nil, c.err
	}
	p, err := discover(ctx, issuer)
	c.issuer, c.provider, c.err = issuer, p, err
	if err != nil {
		c.failed = time.Now()
	}
	return p, err
}

// complete marks the state of a flow as used; false when it was used before or too many flows are kept.
func (c *Client) complete(state string, now time.Time) bool {
	c.usedMu.Lock()
	defer c.usedMu.Unlock()
	if c.used == nil {
		c.used = map[string]time.Time{}
	}
	for s, end := range c.used {
		if now.After(end) {
			delete(c.used, s)
		}
	}
	if _, ok := c.used[state]; ok || len(c.used) >= maxUsedFlows {
		return false
	}
	c.used[state] = now.Add(FlowLifetime)
	return true
}

func oidcConfig(cfg Config) *oidc.Config { return &oidc.Config{ClientID: cfg.ClientID} }

func oauthConfig(p *oidc.Provider, cfg Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.OIDCSettings.RedirectURL,
		Endpoint: p.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

// Start begins a sign-in: it returns the URL of the identity provider's sign-in page and the flow the browser
// keeps until the callback.
func (c *Client) Start(ctx context.Context, cfg Config, next string) (string, Flow, error) {
	p, err := c.providerFor(ctx, cfg.IssuerURL)
	if err != nil {
		return "", Flow{}, err
	}
	flow := Flow{State: rand.Text(), Nonce: rand.Text(), Verifier: oauth2.GenerateVerifier(), Next: next}
	authURL := oauthConfig(p, cfg).AuthCodeURL(
		flow.State, oidc.Nonce(flow.Nonce), oauth2.S256ChallengeOption(flow.Verifier),
	)
	return authURL, flow, nil
}

// Finish completes a sign-in with the state and code of the callback (once per flow): it exchanges the code for
// the tokens, verifies the ID token (signature, issuer, audience, expiry, nonce) and returns the account it
// describes.
func (c *Client) Finish(ctx context.Context, cfg Config, flow Flow, state, code string) (Account, error) {
	if flow.State == "" || subtle.ConstantTimeCompare([]byte(state), []byte(flow.State)) != 1 ||
		!c.complete(flow.State, time.Now()) {
		return Account{}, ErrFlow
	}
	p, err := c.providerFor(ctx, cfg.IssuerURL)
	if err != nil {
		return Account{}, err
	}
	tok, err := oauthConfig(p, cfg).Exchange(
		oidc.ClientContext(ctx, httpClient), code, oauth2.VerifierOption(flow.Verifier),
	)
	if err != nil {
		return Account{}, fmt.Errorf("exchanging the code: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return Account{}, errors.New("the identity provider sent no ID token")
	}
	idt, err := p.Verifier(oidcConfig(cfg)).Verify(ctx, raw)
	if err != nil {
		return Account{}, fmt.Errorf("verifying the ID token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idt.Nonce), []byte(flow.Nonce)) != 1 {
		return Account{}, ErrFlow
	}
	var claims map[string]any
	if err := idt.Claims(&claims); err != nil {
		return Account{}, err
	}
	return accountOf(cfg.OIDCSettings, v1alpha1.OIDCIdentity{Issuer: idt.Issuer, Subject: idt.Subject}, claims)
}

// accountOf reads the account from the claims of an ID token; the groups decide the role. Without access the
// account is returned with the error, for the log.
func accountOf(s v1alpha1.OIDCSettings, id v1alpha1.OIDCIdentity, claims map[string]any) (Account, error) {
	acc := Account{Identity: id}
	// Taken as it is: lower-casing would make "Admin" and "admin" of the identity provider the same account.
	acc.Username, _ = claims[s.UsernameClaim].(string)
	acc.DisplayName, _ = claims["name"].(string)
	acc.Email, _ = claims["email"].(string)
	groups := stringsOf(claims[s.GroupsClaim])
	switch {
	case slices.Contains(groups, s.AdminGroup):
		acc.Role = v1alpha1.RoleAdmin
	case slices.Contains(groups, s.UserGroup):
		acc.Role = v1alpha1.RoleUser
	default:
		return acc, ErrNoAccess
	}
	return acc, nil
}

// stringsOf reads a claim that is a list of strings or a single string.
func stringsOf(v any) []string {
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
