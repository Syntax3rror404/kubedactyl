package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
)

// createToken creates an API token with a session and returns its value and ID.
func (h *harness) createToken(session, name string) (string, string) {
	h.t.Helper()
	code, body := h.do("POST", "/api/auth/tokens", session, map[string]any{"name": name, "expiresInDays": 30})
	expect(h.t, "create token "+name, code, 201, body)
	var created CreatedToken
	_ = json.Unmarshal([]byte(body), &created)
	return created.Token, created.ID
}

// TestPasswordChangesRevokeTokens: a password change revokes the API tokens when asked to, an
// administrator's reset always does, and a token cannot create further tokens.
func TestPasswordChangesRevokeTokens(t *testing.T) {
	h := newHarness(t)
	keep, _ := h.createToken(h.login("alice"), "keep")
	change := func(current, next string, revoke bool) {
		t.Helper()
		session := h.loginWith("alice", current)
		code, body := h.do("PUT", "/api/auth/password", session,
			map[string]any{"current": current, "new": next, "revokeTokens": revoke})
		expect(t, "change password", code, 204, body)
	}
	change("alice-password", "second-password", false)
	code, body := h.do("GET", "/api/servers", keep, nil)
	expect(t, "token after a change without revoking", code, 200, body)

	code, body = h.do("POST", "/api/auth/tokens", keep, map[string]any{"name": "more", "expiresInDays": 30})
	expect(t, "token creates a token", code, 403, body)

	change("second-password", "third-password", true)
	code, body = h.do("GET", "/api/servers", keep, nil)
	expect(t, "token after a change with revoking", code, 401, body)

	reset, _ := h.createToken(h.loginWith("alice", "third-password"), "reset")
	code, body = h.do("PATCH", "/api/users/alice", h.login("admin"), map[string]any{"password": "reset-password"})
	expect(t, "admin reset", code, 200, body)
	code, body = h.do("GET", "/api/servers", reset, nil)
	expect(t, "token after an admin reset", code, 401, body)
	stored := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "alice"}, stored)
	if len(stored.Spec.Tokens) != 0 {
		t.Errorf("tokens kept after the reset: %d", len(stored.Spec.Tokens))
	}
}

// loginWith signs in with a password and returns the session token.
func (h *harness) loginWith(user, password string) string {
	h.t.Helper()
	code, body := h.do("POST", "/api/auth/login", "", map[string]string{"username": user, "password": password})
	expect(h.t, "login "+user, code, 200, body)
	var res LoginResponse
	_ = json.Unmarshal([]byte(body), &res)
	return res.Token
}

// TestLoginThrottlingPerAccount: failures from many addresses (or a spoofed X-Forwarded-For)
// still count for the account, but cannot lock the user out of a browser they signed in with before.
func TestLoginThrottlingPerAccount(t *testing.T) {
	h := newHarness(t)
	login := func(ip, password string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"username":"bob","password":"`+password+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":1234"
		for _, c := range cookies {
			req.AddCookie(c)
		}
		w := httptest.NewRecorder()
		h.router.ServeHTTP(w, req)
		return w
	}
	var device *http.Cookie
	for _, c := range login("10.3.0.1", "bob-password").Result().Cookies() {
		if c.Name == DeviceCookie {
			device = c
		}
	}
	if device == nil || device.Path != "/api/auth/login" || !device.HttpOnly {
		t.Fatalf("a sign-in must set the device cookie: %+v", device)
	}
	for i := range 30 {
		if w := login(fmt.Sprintf("10.1.0.%d", i), "wrong-password"); w.Code != 401 {
			t.Fatalf("failure %d: %d", i, w.Code)
		}
	}
	if w := login("10.2.0.1", "bob-password"); w.Code != http.StatusTooManyRequests {
		t.Errorf("account throttled from a new address: %d", w.Code)
	}
	if w := login("10.3.0.1", "bob-password", device); w.Code != 200 {
		t.Errorf("a known browser must still sign in: %d", w.Code)
	}
	forged := &http.Cookie{Name: DeviceCookie, Value: "9999999999.forged"}
	if w := login("10.4.0.1", "bob-password", forged); w.Code != http.StatusTooManyRequests {
		t.Errorf("a forged device cookie must not help: %d", w.Code)
	}
}

// TestLoginRejectsHugeCredentials: oversized usernames and passwords are refused before they reach the
// limiters (memory) and the log.
func TestLoginRejectsHugeCredentials(t *testing.T) {
	h := newHarness(t)
	long := strings.Repeat("a", 100_000)
	for range 25 {
		for _, creds := range []map[string]string{
			{"username": long, "password": "wrong-password"}, {"username": "bob", "password": long},
		} {
			code, body := h.do("POST", "/api/auth/login", "", creds)
			expect(t, "oversized credentials", code, 400, body)
		}
	}
	// More refused requests than the client limit allows: none of them was counted.
	h.loginWith("bob", "bob-password")
}

// TestDisabledUserCannotSignIn: the right password of a disabled user is refused like a wrong one.
func TestDisabledUserCannotSignIn(t *testing.T) {
	h := newHarness(t)
	code, body := h.do("PATCH", "/api/users/bob", h.login("admin"), map[string]any{"disabled": true})
	expect(t, "disable", code, 200, body)
	code, body = h.do("POST", "/api/auth/login", "", map[string]string{"username": "bob", "password": "bob-password"})
	if code != 401 || !strings.Contains(body, "invalid username or password") {
		t.Errorf("disabled user: %d %s", code, body)
	}
}

// TestSignInRequiresSameOrigin: sign-in and setup refuse forms and requests of other sites.
func TestSignInRequiresSameOrigin(t *testing.T) {
	h := newHarness(t)
	send := func(path, contentType, site string) int {
		req := httptest.NewRequest("POST", path, strings.NewReader(`{"username":"bob","password":"bob-password"}`))
		req.Header.Set("Content-Type", contentType)
		if site != "" {
			req.Header.Set("Sec-Fetch-Site", site)
		}
		w := httptest.NewRecorder()
		h.router.ServeHTTP(w, req)
		return w.Code
	}
	for _, c := range []struct {
		path, contentType, site string
		want                    int
	}{
		{"/api/auth/login", "application/json", "same-origin", 200},
		{"/api/auth/login", "application/json", "", 200},
		{"/api/auth/login", "application/json", "cross-site", 403},
		{"/api/auth/login", "text/plain", "", 403},
		{"/api/auth/login", "application/x-www-form-urlencoded", "", 403},
		{"/api/setup", "text/plain", "", 403},
		{"/api/setup", "application/json", "cross-site", 403},
	} {
		if code := send(c.path, c.contentType, c.site); code != c.want {
			t.Errorf("%s %s %q: %d, want %d", c.path, c.contentType, c.site, code, c.want)
		}
	}
}

// TestAuditLog: sign-ins, file and user changes are logged with the user, the API token used and
// the changed fields (never a password).
func TestAuditLog(t *testing.T) {
	h := newHarness(t)
	var buf bytes.Buffer
	h.api.Log = slog.New(slog.NewTextHandler(&buf, nil))
	token, id := h.createToken(h.login("admin"), "ci")
	code, body := h.do("PATCH", "/api/users/bob", token,
		map[string]any{"role": "admin", "password": "secret-new-password"})
	expect(t, "update with a token", code, 200, body)
	h.do("POST", "/api/auth/login", "", map[string]string{"username": "bob", "password": "wrong-password"})
	log := buf.String()
	for _, want := range []string{
		`msg="signed in" user=admin`,
		`msg="user updated" by=admin token=` + id + ` user=bob changes="[role=admin password reset]"`,
		`msg="sign-in failed" user=bob`,
	} {
		if !strings.Contains(log, want) {
			t.Errorf("audit log without %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "secret-new-password") {
		t.Error("the audit log contains a password")
	}
}
