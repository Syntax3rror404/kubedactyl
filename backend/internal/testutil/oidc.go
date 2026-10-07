package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
)

// IdP is an OpenID Connect identity provider for tests. Its sign-in page (/auth) signs in at once with the
// claims set by SignInAs and redirects back with a code; the token endpoint checks the client secret and the
// PKCE verifier and returns an ID token with those claims and the nonce of the sign-in.
type IdP struct {
	URL      string
	ClientID string
	Secret   string

	key    *rsa.PrivateKey
	mu     sync.Mutex
	claims map[string]any
	codes  map[string]authRequest
}

type authRequest struct {
	nonce, challenge string
	claims           map[string]any
}

// NoRedirects is an HTTP client that returns redirects instead of following them, like a browser step by step.
var NoRedirects = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// NewIdP starts an identity provider for the client "kubedactyl" with the secret "client-secret".
func NewIdP(t testing.TB) *IdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	idp := &IdP{ClientID: "kubedactyl", Secret: "client-secret", key: key, codes: map[string]authRequest{}}
	pub := oidctest.PublicKey{PublicKey: key.Public(), KeyID: "k", Algorithm: oidc.RS256}
	keys := &oidctest.Server{PublicKeys: []oidctest.PublicKey{pub}}
	mux := http.NewServeMux()
	mux.Handle("/", keys)
	mux.HandleFunc("/auth", idp.authorize)
	mux.HandleFunc("/token", idp.token)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	keys.SetIssuer(srv.URL)
	idp.URL = srv.URL
	return idp
}

// SignInAs sets the claims of the next sign-in (sub, preferred_username, groups, ...).
func (p *IdP) SignInAs(claims map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.claims = claims
}

// IDToken signs an ID token with the claims; iss, aud and exp are added unless set.
func (p *IdP) IDToken(claims map[string]any) string {
	all := map[string]any{"iss": p.URL, "aud": p.ClientID, "exp": time.Now().Add(time.Hour).Unix()}
	maps.Copy(all, claims)
	raw, err := json.Marshal(all)
	if err != nil {
		panic(err)
	}
	return oidctest.SignIDToken(p.key, "k", oidc.RS256, string(raw))
}

func (p *IdP) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != p.ClientID || q.Get("code_challenge_method") != "S256" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	code := rand.Text()
	p.mu.Lock()
	p.codes[code] = authRequest{nonce: q.Get("nonce"), challenge: q.Get("code_challenge"), claims: p.claims}
	p.mu.Unlock()
	back, _ := url.Parse(q.Get("redirect_uri"))
	back.RawQuery = url.Values{"code": {code}, "state": {q.Get("state")}}.Encode()
	http.Redirect(w, r, back.String(), http.StatusFound)
}

func (p *IdP) token(w http.ResponseWriter, r *http.Request) {
	id, secret, _ := r.BasicAuth()
	p.mu.Lock()
	req, ok := p.codes[r.FormValue("code")]
	delete(p.codes, r.FormValue("code"))
	p.mu.Unlock()
	sum := sha256.Sum256([]byte(r.FormValue("code_verifier")))
	pkce := base64.RawURLEncoding.EncodeToString(sum[:]) == req.challenge
	if !ok || id != p.ClientID || secret != p.Secret || !pkce {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	claims := maps.Clone(req.claims)
	claims["nonce"] = req.nonce
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": "at", "token_type": "Bearer", "expires_in": 300, "id_token": p.IDToken(claims),
	})
}
