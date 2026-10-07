package httpapi

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"app/internal/testutil"
)

// oidcHarness turns single sign-on on with a test identity provider (users in "players", admins in "admins").
func oidcHarness(t *testing.T) (*harness, *testutil.IdP) {
	t.Helper()
	h, idp := newHarness(t), testutil.NewIdP(t)
	h.enableOIDC(idp, nil)
	return h, idp
}

// enableOIDC saves the single sign-on settings of the test identity provider with the changes.
func (h *harness) enableOIDC(idp *testutil.IdP, changes map[string]any) {
	h.t.Helper()
	oidc := map[string]any{
		"enabled": true, "issuerUrl": idp.URL, "clientId": idp.ClientID, "adminGroup": "admins",
		"userGroup": "players", "redirectUrl": "https://panel.test/api/auth/oidc/callback",
	}
	maps.Copy(oidc, changes)
	code, body := h.do("PUT", "/api/settings", h.login("admin"), map[string]any{
		"storageClasses": []string{"longhorn"}, "oidc": oidc, "oidcClientSecret": idp.Secret,
	})
	if code != 200 || strings.Contains(body, idp.Secret) || !strings.Contains(body, `"oidcClientSecretSet":true`) {
		h.t.Fatalf("enable single sign-on: %d %s", code, body)
	}
}

// browse sends a browser request to the panel with its cookies.
func (h *harness) browse(target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", target, nil)
	req.Host = "panel.test"
	req.Header.Set("X-Forwarded-Proto", "https")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func cookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name && c.MaxAge >= 0 {
			return c
		}
	}
	return nil
}

// ssoSignIn signs in like a browser: start, the identity provider's page, the callback. It returns where the
// callback leads and the session cookie (nil when none).
func (h *harness) ssoSignIn(next string) (string, *http.Cookie) {
	h.t.Helper()
	start := h.browse("/api/auth/oidc/start?next=" + url.QueryEscape(next))
	flow := cookie(start, oidcSecureCookie)
	if start.Code != http.StatusFound || flow == nil {
		h.t.Fatalf("start: %d %v", start.Code, start.Header())
	}
	if flow.SameSite != http.SameSiteLaxMode || !flow.HttpOnly || !flow.Secure || flow.Path != "/" {
		h.t.Errorf("flow cookie: %+v", flow)
	}
	res, err := testutil.NoRedirects.Get(start.Header().Get("Location"))
	if err != nil {
		h.t.Fatal(err)
	}
	_ = res.Body.Close()
	back, _ := url.Parse(res.Header.Get("Location"))
	if back.Host != "panel.test" || back.Path != oidcPath+"/callback" {
		h.t.Fatalf("identity provider redirected to %s", back)
	}
	done := h.browse(back.RequestURI(), flow)
	return done.Header().Get("Location"), cookie(done, SessionCookie)
}

func TestOIDCSignIn(t *testing.T) {
	h, idp := oidcHarness(t)
	if code, body := h.do("GET", "/api/auth/oidc", "", nil); body != `{"enabled":true,"name":"SSO"}` {
		t.Errorf("sign-in page: %d %s", code, body)
	}

	// A new user of the user group gets an account without password and returns to the page they wanted.
	idp.SignInAs(map[string]any{"sub": "s-carol", "preferred_username": "carol", "groups": []any{"players"}})
	to, session := h.ssoSignIn("/servers/x?tab=files")
	if to != "/servers/x?tab=files" || session == nil {
		t.Fatalf("sign-in led to %q, session %v", to, session)
	}
	var me UserView
	code, body := h.do("GET", "/api/auth/me", session.Value, nil)
	_ = json.Unmarshal([]byte(body), &me)
	if code != 200 || me.Username != "carol" || me.Role != "user" || me.OIDC == nil || me.OIDC.Subject != "s-carol" ||
		me.HasPassword {
		t.Errorf("me: %d %s", code, body)
	}
	code, body = h.do("PATCH", "/api/users/carol", h.login("admin"), map[string]string{"role": "admin"})
	expect(t, "role of a linked account", code, http.StatusConflict, body)

	// An existing account with the same name is used only when linking by username is on. Then alice becomes
	// an administrator through her group, signs in only through single sign-on and her old sessions end.
	idp.SignInAs(map[string]any{"sub": "s-alice", "preferred_username": "alice", "groups": []any{"admins"}})
	if to, session = h.ssoSignIn("/"); to != "/login?sso=account" || session != nil {
		t.Errorf("alice without linking by username: led to %q, session %v", to, session)
	}
	old := h.login("alice")
	h.enableOIDC(idp, map[string]any{"linkByUsername": true})
	if to, session = h.ssoSignIn("//evil.example.com"); to != "/" || session == nil {
		t.Errorf("alice: led to %q, session %v", to, session)
	}
	code, body = h.do("GET", "/api/auth/me", old, nil)
	expect(t, "session from before the link", code, http.StatusUnauthorized, body)
	creds := map[string]string{"username": "alice", "password": "alice-password"}
	code, body = h.do("POST", "/api/auth/login", "", creds)
	expect(t, "password of an account linked without keeping it", code, http.StatusUnauthorized, body)
	code, body = h.do("GET", "/api/users", session.Value, nil)
	expect(t, "alice as administrator", code, 200, body)

	for _, tc := range []struct {
		claims map[string]any
		reason string
	}{
		{map[string]any{"sub": "s-dave", "preferred_username": "dave", "groups": []any{"others"}}, "access"},
		{map[string]any{"sub": "s-mallory", "preferred_username": "alice", "groups": []any{"admins"}}, "account"},
		{map[string]any{"sub": "s-eve", "preferred_username": "Eve Smith", "groups": []any{"players"}}, "account"},
	} {
		idp.SignInAs(tc.claims)
		if to, session := h.ssoSignIn("/"); to != "/login?sso="+tc.reason || session != nil {
			t.Errorf("%v: led to %q, session %v", tc.claims, to, session)
		}
	}
}

// TestOIDCRefusesForeignCallbacks: a callback without the flow this browser started (login CSRF), with an
// error of the identity provider or while single sign-on is off starts no session.
func TestOIDCRefusesForeignCallbacks(t *testing.T) {
	h, idp := oidcHarness(t)
	idp.SignInAs(map[string]any{"sub": "s-carol", "preferred_username": "carol", "groups": []any{"players"}})
	start := h.browse("/api/auth/oidc/start")
	forged := &http.Cookie{Name: oidcSecureCookie, Value: "e30.forged"}
	for label, w := range map[string]*httptest.ResponseRecorder{
		"no flow":           h.browse(oidcPath + "/callback?code=c&state=s"),
		"forged flow":       h.browse(oidcPath+"/callback?code=c&state=s", forged),
		"wrong state":       h.browse(oidcPath+"/callback?code=c&state=s", cookie(start, oidcSecureCookie)),
		"provider declined": h.browse(oidcPath+"/callback?error=access_denied", cookie(start, oidcSecureCookie)),
	} {
		if loc := w.Header().Get("Location"); loc != "/login?sso=failed" || cookie(w, SessionCookie) != nil {
			t.Errorf("%s: led to %q", label, loc)
		}
	}

	code, body := h.do("PUT", "/api/settings", h.login("admin"), map[string]any{"storageClasses": []string{"longhorn"}})
	expect(t, "turn single sign-on off", code, 200, body)
	if loc := h.browse("/api/auth/oidc/start").Header().Get("Location"); loc != "/login?sso=failed" {
		t.Errorf("start while off: %q", loc)
	}
	code, body = h.do("GET", "/api/settings", h.login("admin"), nil)
	if !strings.Contains(body, `"oidcClientSecretSet":true`) {
		t.Errorf("the client secret is kept when it is not sent: %d %s", code, body)
	}
	_, body = h.do("GET", "/api/settings", h.login("alice"), nil)
	if !strings.Contains(body, `"oidcClientSecretSet":false`) {
		t.Errorf("users learn whether a client secret is stored: %s", body)
	}
}

func TestOIDCSettingsValidation(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	valid := func(change map[string]any) map[string]any {
		oidc := map[string]any{
			"enabled": true, "issuerUrl": "https://idp.test", "clientId": "c", "adminGroup": "a", "userGroup": "u",
			"redirectUrl": "https://panel.test/api/auth/oidc/callback",
		}
		maps.Copy(oidc, change)
		return map[string]any{"storageClasses": []string{"longhorn"}, "oidc": oidc}
	}
	for label, settings := range map[string]map[string]any{
		"no client":            valid(map[string]any{"clientId": ""}),
		"no redirect URL":      valid(map[string]any{"redirectUrl": ""}),
		"other redirect path":  valid(map[string]any{"redirectUrl": "https://panel.test/callback"}),
		"same groups":          valid(map[string]any{"userGroup": "a"}),
		"plain http":           valid(map[string]any{"issuerUrl": "http://idp.test"}),
		"no identity provider": valid(map[string]any{"issuerUrl": "http://127.0.0.1:1"}),
	} {
		code, body := h.do("PUT", "/api/settings", admin, settings)
		if code != 422 || !strings.Contains(body, `"oidc.`) {
			t.Errorf("%s: %d %s", label, code, body)
		}
	}
}

func TestLocalPath(t *testing.T) {
	for next, want := range map[string]string{
		"/servers/a?tab=files":                 "/servers/a?tab=files",
		"":                                     "/",
		"//evil.example.com":                   "/",
		"/\\evil.example.com":                  "/",
		"https://evil.example":                 "/",
		"/api/servers/a/files/download?file=x": "/",
		"servers":                              "/",
		"/a\nb":                                "/",
	} {
		if got := localPath(next); got != want {
			t.Errorf("localPath(%q) = %q, want %q", next, got, want)
		}
	}
}
