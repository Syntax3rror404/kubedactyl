package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/tenancy"
)

// addSubuser lets alice invite carol and returns carol's session.
func (h *harness) addSubuser(alice string) string {
	h.t.Helper()
	code, body := h.do("POST", "/api/invites", alice, map[string]any{"username": "carol", "role": "admin"})
	if code != 201 {
		h.t.Fatalf("invite: %d %s", code, body)
	}
	var invite CreatedInvite
	_ = json.Unmarshal([]byte(body), &invite)
	if invite.Owner != "alice" || invite.Role != v1alpha1.RoleUser {
		h.t.Errorf("a user invites subusers with role user: %s", body)
	}
	code, body = h.do("POST", "/api/auth/invite", "",
		map[string]string{"token": invite.Token, "password": "carol-password"})
	if code != 201 {
		h.t.Fatalf("accept: %d %s", code, body)
	}
	return h.login("carol")
}

func (h *harness) share(token string, shares ...v1alpha1.ServerShare) (int, string) {
	return h.do("PATCH", "/api/servers/alice-srv", token, map[string]any{"sharedWith": shares})
}

func TestSubuserSeesOnlySharedServers(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	carol := h.addSubuser(alice)

	code, body := h.do("GET", "/api/servers", carol, nil)
	if code != 200 || strings.Contains(body, "-srv") {
		t.Errorf("nothing is shared yet: %d %s", code, body)
	}
	code, body = h.share(h.login("bob"), v1alpha1.ServerShare{User: "carol"})
	expect(t, "share another user's server", code, 404, body)
	code, body = h.share(alice, v1alpha1.ServerShare{User: "bob"})
	expect(t, "share with a user who is no subuser", code, 422, body)
	code, body = h.share(alice, v1alpha1.ServerShare{User: "carol"})
	expect(t, "share", code, 200, body)

	code, body = h.do("GET", "/api/servers", carol, nil)
	if code != 200 || !strings.Contains(body, "alice-srv") || strings.Contains(body, "bob-srv") ||
		strings.Contains(body, `"VIS"`) {
		t.Errorf("the shared server without its variables: %d %s", code, body)
	}
	for _, r := range []struct{ method, path string }{
		{"POST", "/api/servers/alice-srv/power"}, {"POST", "/api/servers/alice-srv/command"},
		{"PATCH", "/api/servers/alice-srv"}, {"GET", "/api/servers/alice-srv/files/list"},
		{"GET", "/api/servers/alice-srv/schedules"}, {"GET", "/api/servers/alice-srv/diagnostics"},
		{"POST", "/api/servers/alice-srv/backups"}, {"GET", "/api/users"}, {"POST", "/api/invites"},
	} {
		code, body := h.do(r.method, r.path, carol, map[string]string{})
		expect(t, "view only "+r.method+" "+r.path, code, 403, body)
	}
	code, body = h.do("GET", "/api/servers/bob-srv", carol, nil)
	expect(t, "a server that is not shared", code, 404, body)

	h.share(alice, v1alpha1.ServerShare{User: "carol", Access: v1alpha1.ShareControl})
	code, body = h.do("POST", "/api/servers/alice-srv/power", carol, map[string]string{"signal": "start"})
	if code == 403 {
		t.Errorf("control allows power actions: %d %s", code, body)
	}
	console := &consoleSession{api: h.api, token: carol, namespace: tenancy.Namespace("alice"), server: "alice-srv"}
	if _, level := console.authorize(t.Context()); level != accessControl {
		t.Errorf("console access %d, want control", level)
	}
}

func TestSubuserAccounts(t *testing.T) {
	h := newHarness(t)
	alice, admin := h.login("alice"), h.login("admin")
	h.addSubuser(alice)

	carol := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "carol"}, carol)
	if carol.Spec.Owner != "alice" || len(carol.OwnerReferences) != 1 || carol.OwnerReferences[0].Name != "alice" {
		t.Errorf("a subuser belongs to its owner: %+v %+v", carol.Spec, carol.OwnerReferences)
	}
	code, body := h.do("GET", "/api/users", alice, nil)
	if code != 200 || !strings.Contains(body, `"carol"`) || strings.Contains(body, `"bob"`) {
		t.Errorf("users list their subusers: %d %s", code, body)
	}
	code, body = h.do("DELETE", "/api/users/bob", alice, nil)
	expect(t, "delete another account", code, 404, body)
	code, body = h.do("PATCH", "/api/users/carol", admin, map[string]string{"role": "admin"})
	expect(t, "make a subuser an administrator", code, 422, body)
	code, body = h.do("POST", "/api/servers", admin, map[string]any{
		"displayName": "Carols Server", "egg": "paper", "owner": "carol", "memoryMiB": 1024, "diskMiB": 2048,
		"ports": []int{25565}, "environment": map[string]string{"VIS": "x"},
	})
	expect(t, "a subuser owns a server", code, 422, body)

	panel := &v1alpha1.PanelSettings{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: v1alpha1.SettingsName}, panel)
	one := int32(1)
	panel.Spec.SubusersPerUser = &one
	_ = h.client.Update(t.Context(), panel)
	h.api.Settings.Forget()
	code, body = h.do("POST", "/api/invites", alice, map[string]any{})
	expect(t, "invite over the limit", code, 409, body)

	code, body = h.do("DELETE", "/api/users/carol", alice, nil)
	expect(t, "delete an own subuser", code, 204, body)
}

func TestInviteSharesServersAndResetsPasswords(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	code, body := h.do("POST", "/api/invites", alice, map[string]any{
		"username": "dave", "shares": []map[string]string{{"server": "bob-srv"}},
	})
	expect(t, "share another user's server", code, 422, body)
	code, body = h.do("POST", "/api/invites", alice, map[string]any{
		"username": "dave", "shares": []map[string]string{{"server": "alice-srv", "access": "control"}},
	})
	var invite CreatedInvite
	_ = json.Unmarshal([]byte(body), &invite)
	h.do("POST", "/api/auth/invite", "", map[string]string{"token": invite.Token, "password": "dave-password"})
	dave := h.login("dave")
	console := &consoleSession{api: h.api, token: dave, namespace: tenancy.Namespace("alice"), server: "alice-srv"}
	if _, level := console.authorize(t.Context()); level != accessControl {
		t.Errorf("the invite shares its servers: access %d (%d %s)", level, code, body)
	}

	code, body = h.do("POST", "/api/invites", alice, map[string]any{"username": "bob", "reset": true})
	expect(t, "reset another user's password", code, 404, body)
	code, body = h.do("POST", "/api/invites", alice, map[string]any{"username": "dave", "reset": true})
	expect(t, "reset link of an own subuser", code, 201, body)
	_ = json.Unmarshal([]byte(body), &invite)
	code, body = h.do("POST", "/api/auth/invite", "",
		map[string]string{"token": invite.Token, "password": "new-password"})
	expect(t, "set the new password", code, 201, body)
	code, _ = h.do("GET", "/api/auth/me", dave, nil)
	expect(t, "the old session ends", code, 401, "")
	h.loginWith("dave", "new-password")
}

func TestInviteListsNoSharesAsEmpty(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	h.do("POST", "/api/invites", alice, map[string]any{})
	if _, body := h.do("GET", "/api/invites", alice, nil); !strings.Contains(body, `"shares":[]`) {
		t.Errorf("shares must be a list for the frontend: %s", body)
	}
}
