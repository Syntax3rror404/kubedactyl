package httpapi

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
)

// invite creates an invite as admin and returns the token of its link.
func (h *harness) invite(admin string, req map[string]string) CreatedInvite {
	h.t.Helper()
	code, body := h.do("POST", "/api/invites", admin, req)
	expect(h.t, "create invite", code, 201, body)
	var created CreatedInvite
	_ = json.Unmarshal([]byte(body), &created)
	return created
}

func TestInviteCreatesAccountOnce(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	created := h.invite(admin, map[string]string{"note": "for Erin"})

	code, body := h.do("GET", "/api/auth/invite?token="+url.QueryEscape(created.Token), "", nil)
	expect(t, "check invite", code, 200, body)
	accept := map[string]string{"token": created.Token, "username": "erin", "password": "short"}
	code, body = h.do("POST", "/api/auth/invite", "", accept)
	expect(t, "short password", code, 422, body)
	accept["password"] = "erin-password"
	code, body = h.do("POST", "/api/auth/invite", "", accept)
	expect(t, "accept invite", code, 201, body)
	var res LoginResponse
	_ = json.Unmarshal([]byte(body), &res)
	if res.User.Username != "erin" || res.User.Role != v1alpha1.RoleUser {
		t.Errorf("account: %+v", res.User)
	}
	code, body = h.do("GET", "/api/servers", res.Token, nil)
	expect(t, "new session", code, 200, body)

	accept["username"] = "erin2"
	code, body = h.do("POST", "/api/auth/invite", "", accept)
	expect(t, "second use", code, 404, body)
	code, body = h.do("GET", "/api/invites", admin, nil)
	if code != 200 || body != "[]" {
		t.Errorf("invites after use: %d %s", code, body)
	}
}

func TestInviteWithUsername(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("POST", "/api/invites", admin, map[string]string{"username": "alice"})
	expect(t, "taken username", code, 422, body)
	created := h.invite(admin, map[string]string{"username": "Frank", "role": "admin"})
	if created.Username != "frank" {
		t.Errorf("username not normalized: %q", created.Username)
	}
	code, body = h.do("GET", "/api/auth/invite?token="+url.QueryEscape(created.Token), "", nil)
	if code != 200 || !strings.Contains(body, `"username":"frank"`) {
		t.Errorf("details: %d %s", code, body)
	}
	code, body = h.do("POST", "/api/auth/invite", "", map[string]string{
		"token": created.Token, "username": "other", "password": "frank-password",
	})
	expect(t, "accept", code, 201, body)
	u := &v1alpha1.User{}
	if err := h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "frank"}, u); err != nil ||
		u.Spec.Role != v1alpha1.RoleAdmin {
		t.Errorf("frank: %v %+v", err, u.Spec)
	}
}

func TestInvalidInvites(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	created := h.invite(admin, map[string]string{})
	for _, token := range []string{"garbage", created.ID + ".wrong-secret", "UPPER.case"} {
		code, body := h.do("GET", "/api/auth/invite?token="+url.QueryEscape(token), "", nil)
		expect(t, "token "+token, code, 404, body)
	}
	invite := &v1alpha1.Invite{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: created.ID}, invite)
	invite.Spec.ExpiresAt = metav1.NewTime(time.Now().Add(-time.Minute))
	_ = h.client.Update(t.Context(), invite)
	code, body := h.do("GET", "/api/auth/invite?token="+url.QueryEscape(created.Token), "", nil)
	expect(t, "expired", code, 404, body)
	code, body = h.do("POST", "/api/invites/"+created.ID+"/renew", admin, nil)
	expect(t, "renew", code, 200, body)
	var renewed CreatedInvite
	_ = json.Unmarshal([]byte(body), &renewed)
	if renewed.Expired || renewed.Token == created.Token || !renewed.ExpiresAt.After(time.Now().Add(6*24*time.Hour)) {
		t.Errorf("renewed: %+v", renewed)
	}
	code, body = h.do("GET", "/api/auth/invite?token="+url.QueryEscape(renewed.Token), "", nil)
	expect(t, "renewed link", code, 200, body)

	revoked := h.invite(admin, map[string]string{})
	code, body = h.do("DELETE", "/api/invites/"+revoked.ID, admin, nil)
	expect(t, "revoke", code, 204, body)
	code, body = h.do("GET", "/api/auth/invite?token="+url.QueryEscape(revoked.Token), "", nil)
	expect(t, "revoked", code, 404, body)
	code, body = h.do("GET", "/api/auth/invite?token="+url.QueryEscape(h.invite(admin, nil).Token), "", nil)
	expect(t, "five failures from one client", code, 429, body)
}

func TestMustChangePassword(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("POST", "/api/users", admin, map[string]any{
		"username": "gina", "password": "gina-password", "mustChangePassword": true,
	})
	expect(t, "create", code, 201, body)
	gina := h.login("gina")
	code, body = h.do("GET", "/api/auth/me", gina, nil)
	if code != 200 || !strings.Contains(body, `"mustChangePassword":true`) {
		t.Errorf("me: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/servers", gina, nil)
	expect(t, "servers before the change", code, 403, body)
	code, body = h.do("PUT", "/api/auth/password", gina, map[string]string{
		"current": "gina-password", "new": "gina-own-password",
	})
	expect(t, "change", code, 204, body)
	code, body = h.do("POST", "/api/auth/login", "", map[string]string{
		"username": "gina", "password": "gina-own-password",
	})
	expect(t, "login", code, 200, body)
	var res LoginResponse
	_ = json.Unmarshal([]byte(body), &res)
	code, body = h.do("GET", "/api/servers", res.Token, nil)
	expect(t, "servers after the change", code, 200, body)

	code, body = h.do("PATCH", "/api/users/gina", admin, map[string]bool{"mustChangePassword": true})
	expect(t, "require again", code, 200, body)
	code, body = h.do("GET", "/api/servers", res.Token, nil)
	expect(t, "servers after the admin required a change", code, 403, body)
}
