package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/console"
	"app/internal/diagnostics"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
	"app/internal/schedule"
	"app/internal/serverctl"
	"app/internal/settings"
	"app/internal/tenancy"
	"app/internal/testutil"
	"app/internal/users"
)

const sysNS = testutil.Namespace

type harness struct {
	t      *testing.T
	router *gin.Engine
	client client.Client
	api    *API
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	hash := func(pw string) string { h, _ := auth.HashPassword(pw); return h }
	user := func(name string, role v1alpha1.UserRole) *v1alpha1.User {
		return &v1alpha1.User{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: sysNS},
			Spec:       v1alpha1.UserSpec{Role: role, PasswordHash: hash(name + "-password")},
		}
	}
	egg := &v1alpha1.Egg{ObjectMeta: metav1.ObjectMeta{Name: "paper", Namespace: sysNS}, Spec: v1alpha1.EggSpec{
		DisplayName:  "Paper",
		Startup:      "java",
		DockerImages: []v1alpha1.DockerImage{{Name: "Java 21", Image: "img:21"}, {Name: "Java 17", Image: "img:17"}},
		Install:      v1alpha1.InstallScript{Script: "secret install script"},
		Variables: []v1alpha1.EggVariable{
			{EnvVariable: "VIS", UserViewable: true, UserEditable: true, Rules: "required|string"},
			{EnvVariable: "RO", UserViewable: true, UserEditable: false},
			{EnvVariable: "HID", UserViewable: false, UserEditable: false},
		},
	}}
	server := func(name, owner string) *v1alpha1.GameServer {
		return &v1alpha1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: tenancy.Namespace(owner),
				Labels:    map[string]string{tenancy.LabelOwner: owner},
			},
			Spec: v1alpha1.GameServerSpec{EggRef: "paper", Image: "img:21", Ports: []int32{25565},
				Resources:   v1alpha1.Resources{MemoryMiB: 1024, DiskMiB: 2048},
				Environment: map[string]string{"VIS": "a", "RO": "b", "HID": "c"}},
		}
	}
	aliceSrv := server("alice-srv", "alice")
	aliceSrv.Spec.LoadBalancerPool = "general-pool"
	aliceSrv.Status.Address = "10.0.0.5"
	panel := &v1alpha1.PanelSettings{
		ObjectMeta: metav1.ObjectMeta{Name: v1alpha1.SettingsName, Namespace: sysNS},
		Spec: v1alpha1.PanelSettingsSpec{
			ExternalDomain: "play.example.com", StorageClasses: []string{"longhorn"}, DefaultStorageClass: "longhorn",
			LoadBalancerPools: []string{"general-pool", "other-pool"}, DefaultLoadBalancerPool: "general-pool",
		},
	}
	c := testutil.Builder(t).
		WithStatusSubresource(&v1alpha1.User{}, &v1alpha1.GameServer{}).
		WithIndex(&v1alpha1.GameServer{}, IndexServerName, ServerNameIndex).
		WithObjects(user("admin", v1alpha1.RoleAdmin), user("alice", v1alpha1.RoleUser), user("bob", v1alpha1.RoleUser),
			egg, aliceSrv, server("bob-srv", "bob"), panel,
			&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "longhorn"}, Provisioner: "driver.longhorn.io"},
			&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "fast"}, Provisioner: "driver.longhorn.io"},
			// a server of another installation in the same cluster (namespace other-user-eve)
			&v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: "foreign-srv", Namespace: "other-user-eve"},
				Spec:       v1alpha1.GameServerSpec{EggRef: "paper"},
			}).
		Build()
	signer, _ := auth.NewSigner(bytes.Repeat([]byte("k"), 32))
	api := &API{
		Client:         c,
		Reader:         c,
		Opts:           gameserver.Options{Namespace: sysNS},
		Log:            testutil.Logger(),
		Settings:       &settings.Store{Client: c, Reader: c, Namespace: sysNS},
		Users:          &users.Store{Client: c, Reader: c, Namespace: sysNS},
		Eggs:           &eggstore.Store{Client: c, Reader: c, Namespace: sysNS},
		Diagnostics:    &diagnostics.Diagnoser{Resolver: noDNS{}, PublicResolver: noDNS{}},
		Files:          &files.Service{Activity: files.NewActivity(), Reader: c, Trigger: func(string, string) {}},
		Ops:            &serverctl.Ops{Client: c, Namespace: sysNS, Hub: console.NewHub(nil, testutil.Logger())},
		Schedules:      &schedule.Runner{Client: c, Reader: c, Location: time.UTC},
		SetupToken:     "setup-token",
		Hub:            console.NewHub(nil, testutil.Logger()),
		Signer:         signer,
		Limiter:        auth.NewLimiter(5, 60e9),
		ClientLimiter:  auth.NewLimiter(20, 60e9),
		AccountLimiter: auth.NewLimiter(30, 60e9),
		Trigger:        func(string, string) {},
	}
	r := gin.New()
	api.Register(r.Group("/api"))
	return &harness{t: t, router: r, client: c, api: api}
}

// do sends a request with an optional bearer token and returns status and body.
func (h *harness) do(method, path, token string, body any) (int, string) {
	h.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func (h *harness) login(user string) string {
	h.t.Helper()
	code, body := h.do(
		"POST", "/api/auth/login", "", map[string]string{"username": user, "password": user + "-password"},
	)
	if code != 200 {
		h.t.Fatalf("login %s: %d %s", user, code, body)
	}
	var res LoginResponse
	_ = json.Unmarshal([]byte(body), &res)
	return res.Token
}

func expect(t *testing.T, label string, got, want int, body string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: status %d, want %d (%s)", label, got, want, body)
	}
}

func TestAuthentication(t *testing.T) {
	h := newHarness(t)
	code, body := h.do("GET", "/api/servers", "", nil)
	expect(t, "no credentials", code, 401, body)
	code, body = h.do(
		"POST", "/api/auth/login", "", map[string]string{"username": "alice", "password": "wrong-password"},
	)
	expect(t, "wrong password", code, 401, body)
	code, body = h.do("POST", "/api/auth/login", "", map[string]string{"username": "nobody", "password": "whatever-pw"})
	expect(t, "unknown user", code, 401, body)
	code, body = h.do("GET", "/api/auth/me", "garbage.token", nil)
	expect(t, "forged token", code, 401, body)

	tok := h.login("alice")
	code, body = h.do("GET", "/api/auth/me", tok, nil)
	expect(t, "me", code, 200, body)
	if !strings.Contains(body, `"username":"alice"`) || strings.Contains(body, "argon2") {
		t.Errorf("me leaks or misses data: %s", body)
	}
}

func TestUserSeesOnlyOwnServers(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	code, body := h.do("GET", "/api/servers", alice, nil)
	expect(t, "list", code, 200, body)
	if !strings.Contains(body, "alice-srv") || strings.Contains(body, "bob-srv") {
		t.Errorf("alice sees wrong servers: %s", body)
	}
	for _, path := range []string{
		"/api/servers/bob-srv", "/api/servers/bob-srv/stats", "/api/servers/bob-srv/diagnostics",
		"/api/servers/bob-srv/files/list",
	} {
		code, body = h.do("GET", path, alice, nil)
		expect(t, "foreign "+path, code, 404, body)
	}
	code, body = h.do("POST", "/api/servers/bob-srv/power", alice, map[string]string{"signal": "stop"})
	expect(t, "foreign power", code, 404, body)
	// Every other write on a foreign server answers 404 as well (before its body is looked at).
	for _, r := range []struct{ method, path string }{
		{"POST", "/command"}, {"POST", "/reinstall"}, {"PATCH", ""}, {"PUT", "/schedules"}, {"POST", "/backups"},
		{"POST", "/backups/x.tar.gz/restore"}, {"DELETE", "/backups/x.tar.gz"},
		{"POST", "/files/write?file=/x"}, {"POST", "/files/create-folder"}, {"POST", "/files/delete"},
		{"PUT", "/files/rename"}, {"POST", "/files/upload"}, {"POST", "/files/compress"},
		{"POST", "/files/decompress"}, {"POST", "/files/pull"}, {"GET", "/files/contents?file=/x"},
		{"GET", "/files/download?file=/x"},
	} {
		code, body = h.do(r.method, "/api/servers/bob-srv"+r.path, alice, map[string]string{})
		expect(t, "foreign "+r.method+" "+r.path, code, 404, body)
	}
	for _, method := range []string{"GET", "POST"} {
		code, body = h.do(method, "/api/servers/bob-srv/files/session", alice, nil)
		expect(t, "foreign files session "+method, code, 404, body)
	}
	code, body = h.do("GET", "/api/servers/alice-srv/files/session", alice, nil)
	if code != 200 || !strings.Contains(body, `"state":"Stopped"`) ||
		!strings.Contains(body, `"idleTimeoutSeconds":60`) {
		t.Errorf("own files session: %d %s", code, body)
	}

	code, body = h.do("GET", "/api/servers/alice-srv/diagnostics", alice, nil)
	expect(t, "own diagnostics", code, 200, body)
	if strings.Contains(body, "10.0.0.5") || !strings.Contains(body, `"id":"dns-a"`) {
		t.Errorf("diagnostics must run the DNS checks and hide the load balancer IP from users: %s", body)
	}

	code, body = h.do("GET", "/api/servers/alice-srv", alice, nil)
	expect(t, "own server", code, 200, body)
	if strings.Contains(body, `"HID"`) || !strings.Contains(body, `"VIS"`) {
		t.Errorf("hidden variable visible to user: %s", body)
	}
	code, body = h.do("GET", "/api/eggs/paper", alice, nil)
	expect(t, "egg", code, 200, body)
	if strings.Contains(body, "HID") || strings.Contains(body, "secret install script") {
		t.Errorf("egg leaks hidden data to user: %s", body)
	}

	admin := h.login("admin")
	code, body = h.do("GET", "/api/servers", admin, nil)
	if code != 200 || !strings.Contains(body, "alice-srv") || !strings.Contains(body, "bob-srv") {
		t.Errorf("admin must see all servers: %d %s", code, body)
	}
	if strings.Contains(body, "foreign-srv") {
		t.Errorf("admin sees a server of another installation: %s", body)
	}
	code, body = h.do("GET", "/api/servers/foreign-srv", admin, nil)
	expect(t, "server of another installation", code, 404, body)
	code, body = h.do("GET", "/api/servers/bob-srv", admin, nil)
	if code != 200 || !strings.Contains(body, `"HID"`) {
		t.Errorf("admin must see all variables: %d %s", code, body)
	}
}

func TestUserUpdateRestrictions(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	cases := []struct {
		label string
		body  map[string]any
		want  int
	}{
		{"memory", map[string]any{"memoryMiB": 8192}, 403},
		{"ports", map[string]any{"ports": []int{1}}, 403},
		{"traffic policy", map[string]any{"externalTrafficPolicy": "Cluster"}, 403},
		{"startup", map[string]any{"startup": "rm -rf /"}, 403},
		{"foreign image", map[string]any{"image": "evil:latest"}, 403},
		{"read-only variable", map[string]any{"environment": map[string]string{"RO": "changed"}}, 403},
		{"hidden variable", map[string]any{"environment": map[string]string{"HID": "changed"}}, 403},
		{"editable variable", map[string]any{"environment": map[string]string{"VIS": "new"}}, 200},
		{"egg image", map[string]any{"image": "img:17"}, 200},
		{"display name", map[string]any{"displayName": "Mine"}, 200},
	}
	for _, c := range cases {
		code, body := h.do("PATCH", "/api/servers/alice-srv", alice, c.body)
		expect(t, c.label, code, c.want, body)
	}
	gs := &v1alpha1.GameServer{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: tenancy.Namespace("alice"), Name: "alice-srv"}, gs)
	if gs.Spec.Environment["VIS"] != "new" || gs.Spec.Environment["HID"] != "c" || gs.Spec.Resources.MemoryMiB != 1024 {
		t.Errorf("unexpected spec after updates: %+v", gs.Spec)
	}
}

func TestAdminOnlyEndpoints(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	for _, r := range []struct{ method, path string }{
		{"GET", "/api/cluster"}, {"GET", "/api/cluster/nodes"}, {"POST", "/api/cluster/nodes/probe"},
		{"GET", "/api/cluster/identity"}, {"GET", "/api/cluster/health"}, {"GET", "/api/users"}, {"POST", "/api/users"},
		{"DELETE", "/api/users/bob"}, {"POST", "/api/servers"}, {"DELETE", "/api/servers/alice-srv"},
		{"POST", "/api/servers/alice-srv/transfer"}, {"PUT", "/api/settings"}, {"GET", "/api/upgrade"},
		{"POST", "/api/upgrade"}, {"GET", "/api/versions"}, {"GET", "/api/settings/storage-classes"},
		{"GET", "/api/settings/load-balancer-pools"}, {"POST", "/api/eggs/import-url"}, {"DELETE", "/api/eggs/paper"},
		{"POST", "/api/eggs"}, {"PUT", "/api/eggs/paper"}, {"GET", "/api/eggs/paper/export"},
		{"POST", "/api/eggs/paper/update-from-url"}, {"POST", "/api/eggs/import"}, {"GET", "/api/users/bob"},
		{"PATCH", "/api/users/bob"}, {"GET", "/api/egg-library"}, {"GET", "/api/egg-library/egg"},
		{"GET", "/api/invites"}, {"POST", "/api/invites"}, {"DELETE", "/api/invites/1a2b3c4d5e6f"},
		{"POST", "/api/invites/1a2b3c4d5e6f/renew"},
	} {
		code, body := h.do(r.method, r.path, alice, map[string]string{})
		expect(t, "user "+r.method+" "+r.path, code, 403, body)
	}
}

func TestAdminSetsTrafficPolicy(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("PATCH", "/api/servers/alice-srv", admin, map[string]any{"externalTrafficPolicy": "Sideways"})
	expect(t, "unknown traffic policy", code, 400, body)
	code, body = h.do("PATCH", "/api/servers/alice-srv", admin, map[string]any{"externalTrafficPolicy": "Cluster"})
	expect(t, "traffic policy", code, 200, body)
	gs := &v1alpha1.GameServer{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: tenancy.Namespace("alice"), Name: "alice-srv"}, gs)
	if gs.Spec.ExternalTrafficPolicy != v1alpha1.TrafficCluster {
		t.Errorf("traffic policy not stored: %q", gs.Spec.ExternalTrafficPolicy)
	}
}

func TestAdminTransferChecks(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("POST", "/api/servers/alice-srv/transfer", admin, map[string]string{"owner": "alice"})
	expect(t, "transfer to the owner", code, 409, body)
	code, body = h.do("POST", "/api/servers/alice-srv/transfer", admin, map[string]string{"owner": "nobody"})
	expect(t, "transfer to an unknown user", code, 422, body)
	code, body = h.do("POST", "/api/servers/alice-srv/transfer", admin, map[string]string{})
	expect(t, "transfer without owner", code, 400, body)
}

func TestAdminCreatesUsersAndServers(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do(
		"POST", "/api/users", admin, map[string]string{"username": "carol", "password": "carol-password"},
	)
	expect(t, "create user", code, 201, body)
	if strings.Contains(body, "argon2") {
		t.Errorf("user view leaks hash: %s", body)
	}
	ns := &corev1.Namespace{}
	if err := h.client.Get(t.Context(), client.ObjectKey{Name: "kubedactyl-user-carol"}, ns); err != nil ||
		ns.Labels[tenancy.LabelUser] != "carol" {
		t.Errorf("namespace not created: %v %v", err, ns.Labels)
	}
	stored := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "carol"}, stored)
	if !strings.HasPrefix(stored.Spec.PasswordHash, "$argon2id$") ||
		strings.Contains(stored.Spec.PasswordHash, "carol-password") {
		t.Errorf("password not hashed: %q", stored.Spec.PasswordHash)
	}
	code, body = h.do("POST", "/api/users", admin, map[string]string{"username": "Bad Name", "password": "long-enough"})
	expect(t, "invalid username", code, 422, body)
	code, body = h.do("POST", "/api/users", admin, map[string]string{"username": "dave", "password": "short"})
	expect(t, "short password", code, 422, body)

	req := map[string]any{
		"displayName": "Carols Server",
		"egg":         "paper",
		"owner":       "carol",
		"memoryMiB":   1024,
		"diskMiB":     2048,
		"ports":       []int{25565},
		"environment": map[string]string{"VIS": "x"},
	}
	code, body = h.do("POST", "/api/servers", admin, req)
	expect(t, "create server for carol", code, 201, body)
	var gs v1alpha1.GameServer
	_ = json.Unmarshal([]byte(body), &gs)
	if gs.Namespace != "kubedactyl-user-carol" || gs.Labels[tenancy.LabelOwner] != "carol" {
		t.Errorf("server placed wrong: ns=%s labels=%v", gs.Namespace, gs.Labels)
	}
	carol := h.login("carol")
	code, body = h.do("GET", "/api/servers", carol, nil)
	if code != 200 || !strings.Contains(body, gs.Name) || strings.Contains(body, "alice-srv") {
		t.Errorf("carol list: %d %s", code, body)
	}
	req["owner"] = "ghost"
	code, body = h.do("POST", "/api/servers", admin, req)
	expect(t, "unknown owner", code, 422, body)
}

func TestLastAdminProtectedAndSessions(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("PATCH", "/api/users/admin", admin, map[string]string{"role": "user"})
	expect(t, "demote last admin", code, 409, body)
	code, body = h.do("PATCH", "/api/users/admin", admin, map[string]bool{"disabled": true})
	expect(t, "disable last admin", code, 409, body)
	code, body = h.do("DELETE", "/api/users/admin", admin, nil)
	expect(t, "delete self", code, 409, body)

	alice := h.login("alice")
	code, body = h.do("PATCH", "/api/users/alice", admin, map[string]bool{"disabled": true})
	expect(t, "disable alice", code, 200, body)
	code, body = h.do("GET", "/api/auth/me", alice, nil)
	expect(t, "disabled user session", code, 401, body)
	code, body = h.do(
		"POST", "/api/auth/login", "", map[string]string{"username": "alice", "password": "alice-password"},
	)
	expect(t, "disabled user login", code, 401, body)

	bob := h.login("bob")
	code, body = h.do(
		"PUT", "/api/auth/password", bob, map[string]string{"current": "wrong-one", "new": "bob-new-password"},
	)
	expect(t, "password change with wrong current", code, 422, body)
	code, body = h.do(
		"PUT", "/api/auth/password", bob, map[string]string{"current": "bob-password", "new": "bob-new-password"},
	)
	expect(t, "password change", code, 204, body)
	code, body = h.do("GET", "/api/auth/me", bob, nil)
	expect(t, "old session after password change", code, 401, body)
}

func TestAPITokens(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	code, body := h.do("POST", "/api/auth/tokens", alice, map[string]any{"name": "ci", "expiresInDays": 30})
	expect(t, "create token", code, 201, body)
	var created CreatedToken
	_ = json.Unmarshal([]byte(body), &created)
	if !strings.HasPrefix(created.Token, "kdt_") {
		t.Fatalf("token: %s", body)
	}
	stored := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "alice"}, stored)
	if strings.Contains(stored.Spec.Tokens[0].Hash, created.Token) || len(stored.Spec.Tokens[0].Hash) != 64 {
		t.Errorf("token stored in clear text")
	}
	code, body = h.do("GET", "/api/servers", created.Token, nil)
	if code != 200 || !strings.Contains(body, "alice-srv") || strings.Contains(body, "bob-srv") {
		t.Errorf("token access: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/users", created.Token, nil)
	expect(t, "user token on admin endpoint", code, 403, body)
	code, body = h.do("DELETE", "/api/auth/tokens/"+created.ID, alice, nil)
	expect(t, "revoke", code, 204, body)
	code, body = h.do("GET", "/api/servers", created.Token, nil)
	expect(t, "revoked token", code, 401, body)
}

func TestTokenLifetimes(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	for _, days := range []int{0, 91} {
		code, body := h.do("POST", "/api/auth/tokens", alice, map[string]any{"name": "ci", "expiresInDays": days})
		expect(t, fmt.Sprintf("token for %d days (default limit 90)", days), code, 422, body)
	}

	// Tokens without an expiry (made before the limit existed) end 90 days after their creation.
	user := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "alice"}, user)
	tokens := map[string]int{}
	for _, age := range []int{10, 100} {
		token, id, hash := auth.NewAPIToken()
		created := metav1.NewTime(time.Now().AddDate(0, 0, -age))
		old := v1alpha1.APIToken{ID: id, Name: "old", Hash: hash, CreatedAt: created}
		user.Spec.Tokens = append(user.Spec.Tokens, old)
		tokens[token] = age
	}
	if err := h.client.Update(t.Context(), user); err != nil {
		t.Fatal(err)
	}
	want := map[int]int{10: 200, 100: 401}
	for token, age := range tokens {
		code, body := h.do("GET", "/api/auth/me", token, nil)
		expect(t, fmt.Sprintf("token without expiry, %d days old", age), code, want[age], body)
	}

	// A shorter session lifetime also ends sessions that were signed before (saved settings apply at once).
	code, body := h.do("PUT", "/api/settings", h.login("admin"),
		map[string]any{"storageClasses": []string{"longhorn"}, "sessionHours": 1})
	expect(t, "shorter session lifetime", code, 200, body)
	old, _ := h.api.Signer.Sign(auth.Session{
		User: "alice", IssuedAt: time.Now().Add(-2 * time.Hour).Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	code, body = h.do("GET", "/api/auth/me", old, nil)
	expect(t, "session older than the session lifetime", code, 401, body)
	code, body = h.do("GET", "/api/auth/me", h.login("alice"), nil)
	expect(t, "new session", code, 200, body)
}

func TestLoginThrottling(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 5; i++ {
		h.do("POST", "/api/auth/login", "", map[string]string{"username": "bob", "password": "wrong-password"})
	}
	code, body := h.do("POST", "/api/auth/login", "", map[string]string{"username": "bob", "password": "bob-password"})
	expect(t, "throttled", code, http.StatusTooManyRequests, body)
}

// TestParallelLoginsAreThrottled: attempts count before the password check, so a burst of parallel
// sign-ins gets no more tries than one after another.
func TestParallelLoginsAreThrottled(t *testing.T) {
	h := newHarness(t)
	var wg sync.WaitGroup
	var checked atomic.Int32
	for range 30 {
		wg.Go(func() {
			code, _ := h.do("POST", "/api/auth/login", "", map[string]string{"username": "bob", "password": "wrong-pw"})
			if code == http.StatusUnauthorized {
				checked.Add(1)
			}
		})
	}
	wg.Wait()
	if n := checked.Load(); n != 5 {
		t.Errorf("%d of 30 parallel attempts were checked, want 5", n)
	}
}

func TestLoginThrottlingAcrossUsernames(t *testing.T) {
	h := newHarness(t)
	// Password spraying: a few tries per name, many names: the client limit stops it.
	for i := range 20 {
		h.do(
			"POST", "/api/auth/login", "",
			map[string]string{"username": fmt.Sprintf("guess-%d", i), "password": "wrong-password"},
		)
	}
	code, body := h.do("POST", "/api/auth/login", "", map[string]string{"username": "bob", "password": "bob-password"})
	expect(t, "client throttled", code, http.StatusTooManyRequests, body)
}

func TestRequestLimitPerUser(t *testing.T) {
	h := newHarness(t)
	h.api.KubeLimiter = kube.NewRateLimiter(1000, 1) // bursts of 2 requests per user
	alice, bob := h.login("alice"), h.login("bob")
	for i := range 2 {
		code, body := h.do("GET", "/api/servers", alice, nil)
		expect(t, fmt.Sprintf("alice request %d", i), code, 200, body)
	}
	code, body := h.do("GET", "/api/servers", alice, nil)
	expect(t, "alice over her limit", code, 429, body)
	code, body = h.do("GET", "/api/servers", bob, nil)
	expect(t, "bob does not notice alice", code, 200, body)
}

func TestRequestRates(t *testing.T) {
	h := newHarness(t)
	h.api.KubeLimiter = kube.NewRateLimiter(1000, 1)
	alice, admin := h.login("alice"), h.login("admin")
	for range 3 { // 2 admitted, 1 refused
		h.do("GET", "/api/servers", alice, nil)
	}
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
	for range 5 { // polling neither counts nor is refused
		code, body := h.do("GET", "/api/request-rates", alice, nil)
		expect(t, "alice's rates", code, 200, body)
		if body != `{"user":{"rate":0.6,"limit":1}}` {
			t.Errorf("alice's rates: %s", body)
		}
	}
	code, body := h.do("GET", "/api/request-rates", admin, nil)
	if code != 200 || !strings.Contains(body, `"panel":{"rate":`) || !strings.Contains(body, `"limit":1000}`) {
		t.Errorf("administrators see the panel: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/request-rates", "", nil)
	expect(t, "no credentials", code, 401, body)
}

func TestBusyPanelAdmitsAdministrators(t *testing.T) {
	h := newHarness(t)
	limiter := kube.NewRateLimiter(5, 100)
	h.api.KubeLimiter = limiter
	alice, admin := h.login("alice"), h.login("admin")
	// 30 calls waiting for the limit of 5 per second: a queue of several seconds.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	for range 30 {
		go func() { _ = limiter.Wait(ctx) }()
	}
	for _, err := limiter.Busy(); err == nil; _, err = limiter.Busy() {
		time.Sleep(10 * time.Millisecond)
	}
	code, body := h.do("GET", "/api/servers", alice, nil)
	expect(t, "users are refused while the panel is busy", code, 503, body)
	code, body = h.do("GET", "/api/servers", admin, nil)
	expect(t, "administrators may still raise the limit", code, 200, body)
}

func TestSettingsStorageAndPools(t *testing.T) {
	h := newHarness(t)
	alice, admin := h.login("alice"), h.login("admin")

	code, body := h.do("GET", "/api/settings", alice, nil)
	if code != 200 || !strings.Contains(body, `"other-pool"`) {
		t.Errorf("user settings: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/servers/alice-srv", alice, nil)
	if code != 200 || !strings.Contains(body, `"address":"play.example.com"`) || strings.Contains(body, "10.0.0.5") {
		t.Errorf("user must see the external domain instead of the IP: %s", body)
	}
	code, body = h.do("GET", "/api/servers", admin, nil)
	if code != 200 || !strings.Contains(body, "10.0.0.5") {
		t.Errorf("admin must see the IP: %s", body)
	}

	code, body = h.do("PATCH", "/api/servers/alice-srv", alice, map[string]string{"loadBalancerPool": "secret-pool"})
	expect(t, "user selects a pool that is not enabled", code, 422, body)
	code, body = h.do("PATCH", "/api/servers/alice-srv", alice, map[string]string{"loadBalancerPool": "other-pool"})
	expect(t, "user selects an enabled pool", code, 200, body)
	gs := &v1alpha1.GameServer{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: tenancy.Namespace("alice"), Name: "alice-srv"}, gs)
	if gs.Spec.LoadBalancerPool != "other-pool" {
		t.Errorf("pool not changed: %q", gs.Spec.LoadBalancerPool)
	}

	req := map[string]any{
		"displayName": "Fast",
		"egg":         "paper",
		"memoryMiB":   1024,
		"diskMiB":     2048,
		"ports":       []int{25565},
		"environment": map[string]string{"VIS": "x"},
	}
	req["storageClass"] = "fast"
	code, body = h.do("POST", "/api/servers", admin, req)
	expect(t, "storage class that is not enabled", code, 422, body)
	delete(req, "storageClass")
	code, body = h.do("POST", "/api/servers", admin, req)
	expect(t, "server with defaults", code, 201, body)
	var created v1alpha1.GameServer
	_ = json.Unmarshal([]byte(body), &created)
	if created.Spec.StorageClass != "longhorn" || created.Spec.LoadBalancerPool != "general-pool" {
		t.Errorf("defaults not applied: %+v", created.Spec)
	}

	for label, bad := range map[string]map[string]any{
		"unknown storage class": {"storageClasses": []string{"missing"}},
		"no storage class":      {"storageClasses": []string{}},
		"invalid domain":        {"storageClasses": []string{"fast"}, "externalDomain": "not a domain"},
	} {
		code, body = h.do("PUT", "/api/settings", admin, bad)
		expect(t, label, code, 422, body)
	}
	h.do("GET", "/api/settings", admin, nil) // the settings are kept in memory from here on
	code, body = h.do(
		"PUT",
		"/api/settings",
		admin,
		map[string]any{
			"storageClasses":      []string{"longhorn", "fast"},
			"defaultStorageClass": "fast",
			"externalDomain":      " Play.Example.org. ",
		},
	)
	if code != 200 || !strings.Contains(body, `"defaultStorageClass":"fast"`) ||
		!strings.Contains(body, `"externalDomain":"play.example.org"`) {
		t.Errorf("save settings: %d %s", code, body)
	}
	// The next reads show the saved settings at once, not the ones kept before.
	code, body = h.do("GET", "/api/settings", admin, nil)
	if !strings.Contains(body, `"externalDomain":"play.example.org"`) {
		t.Errorf("settings after the save: %d %s", code, body)
	}
	if code, body = h.do("GET", "/api/cluster", admin, nil); !strings.Contains(body, `"storageClass":"fast"`) {
		t.Errorf("cluster after the save: %d %s", code, body)
	}
}

// unreachableReader fails every read: the routes that use it must not call the API server.
type unreachableReader struct{ client.Reader }

func (unreachableReader) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return errors.New("the API server must not be asked")
}

func TestSetup(t *testing.T) {
	h := newHarness(t)
	// Public routes: once an administrator exists, they answer from the cache.
	reader := h.api.Users.Reader
	h.api.Users.Reader = unreachableReader{reader}
	code, body := h.do("GET", "/api/setup", "", nil)
	if code != 200 || !strings.Contains(body, `"required":false`) {
		t.Errorf("setup status with an admin: %d %s", code, body)
	}
	code, body = h.do(
		"POST", "/api/setup", "",
		map[string]string{"username": "mallory", "password": "mallory-password", "token": "setup-token"},
	)
	expect(t, "setup with an existing admin", code, 409, body)
	h.api.Users.Reader = reader

	// Without an active administrator the setup page is available exactly once.
	admin := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "admin"}, admin)
	_ = h.client.Delete(t.Context(), admin)
	code, body = h.do("GET", "/api/setup", "", nil)
	if code != 200 || !strings.Contains(body, `"required":true`) {
		t.Errorf("setup status without admin: %d %s", code, body)
	}
	code, body = h.do(
		"POST", "/api/setup", "", map[string]string{"username": "root", "password": "root-password", "token": "wrong"},
	)
	expect(t, "setup with a wrong token", code, 403, body)
	code, body = h.do("POST", "/api/setup", "", map[string]string{"username": "root", "password": "root-password"})
	expect(t, "setup without token", code, 400, body)
	code, body = h.do(
		"POST", "/api/setup", "",
		map[string]string{"username": "Bad Name", "password": "long-enough-pw", "token": "setup-token"},
	)
	expect(t, "setup with invalid username", code, 422, body)
	code, body = h.do(
		"POST", "/api/setup", "", map[string]string{"username": "root", "password": "short", "token": "setup-token"},
	)
	expect(t, "setup with short password", code, 422, body)
	code, body = h.do(
		"POST",
		"/api/setup",
		"",
		map[string]string{
			"username":    "root",
			"password":    "root-password",
			"displayName": "Root",
			"token":       "setup-token",
		},
	)
	expect(t, "setup", code, 201, body)
	var res LoginResponse
	_ = json.Unmarshal([]byte(body), &res)
	if res.User.Role != v1alpha1.RoleAdmin || res.Token == "" {
		t.Errorf("setup must sign in an administrator: %s", body)
	}
	code, body = h.do("GET", "/api/users", res.Token, nil)
	expect(t, "new admin reaches admin endpoints", code, 200, body)
	stored := &v1alpha1.User{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "root"}, stored)
	if !strings.HasPrefix(stored.Spec.PasswordHash, "$argon2id$") {
		t.Errorf("password not hashed: %q", stored.Spec.PasswordHash)
	}
	code, body = h.do(
		"POST", "/api/setup", "",
		map[string]string{"username": "second", "password": "second-password", "token": "setup-token"},
	)
	expect(t, "second setup", code, 409, body)
}

func TestCrossSiteCookieRequests(t *testing.T) {
	h := newHarness(t)
	token := h.login("alice")
	send := func(method, site string, bearer bool) int {
		req := httptest.NewRequest(method, "/api/servers/alice-srv", strings.NewReader(`{"displayName":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		if bearer {
			req.Header.Set("Authorization", "Bearer "+token)
		} else {
			req.AddCookie(&http.Cookie{Name: SessionCookie, Value: token})
		}
		if site != "" {
			req.Header.Set("Sec-Fetch-Site", site)
		}
		w := httptest.NewRecorder()
		h.router.ServeHTTP(w, req)
		return w.Code
	}
	if code := send("PATCH", "cross-site", false); code != 403 {
		t.Errorf("cross-site PATCH with cookie: %d, want 403", code)
	}
	if code := send("PATCH", "same-site", false); code != 403 {
		t.Errorf("same-site (other subdomain) PATCH with cookie: %d, want 403", code)
	}
	if code := send("PATCH", "same-origin", false); code != 200 {
		t.Errorf("same-origin PATCH with cookie: %d, want 200", code)
	}
	if code := send("GET", "cross-site", false); code != 200 {
		t.Errorf("cross-site GET (e.g. a link) must work: %d", code)
	}
	if code := send("PATCH", "cross-site", true); code != 200 {
		t.Errorf("API token requests are not affected: %d", code)
	}
}

// Eggs live in the panel namespace while servers live in user namespaces; the egg's
// file_denylist must still apply to the files of a user's server.
func TestDenylistFromPanelNamespace(t *testing.T) {
	h := newHarness(t)
	egg := &v1alpha1.Egg{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "paper"}, egg)
	egg.Spec.FileDenylist = []string{"server.jar"}
	if err := h.client.Update(t.Context(), egg); err != nil {
		t.Fatal(err)
	}
	gs := &v1alpha1.GameServer{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: tenancy.Namespace("alice"), Name: "alice-srv"}, gs)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	api := &API{Eggs: &eggstore.Store{Client: h.client, Reader: h.client, Namespace: sysNS}}
	if got := api.denylist(c, gs); len(got) != 1 || got[0] != "server.jar" {
		t.Errorf("denylist = %v", got)
	}
}

func TestLogoutEverywhereAndNotice(t *testing.T) {
	h := newHarness(t)
	first, second := h.login("bob"), h.login("bob")
	code, body := h.do("POST", "/api/auth/tokens", first, map[string]any{"name": "ci", "expiresInDays": 7})
	expect(t, "create token", code, 201, body)
	var created CreatedToken
	_ = json.Unmarshal([]byte(body), &created)
	code, body = h.do("POST", "/api/auth/logout-all", first, nil)
	expect(t, "logout everywhere", code, 204, body)
	for label, tok := range map[string]string{"this session": first, "other session": second} {
		code, body = h.do("GET", "/api/auth/me", tok, nil)
		expect(t, label+" after logout everywhere", code, 401, body)
	}
	code, body = h.do("GET", "/api/auth/me", created.Token, nil)
	expect(t, "API tokens stay valid", code, 200, body)

	admin := h.login("admin")
	code, body = h.do(
		"PUT", "/api/settings", admin,
		map[string]any{"storageClasses": []string{"longhorn"}, "serverNotice": strings.Repeat("x", 2001)},
	)
	expect(t, "too long notice", code, 422, body)
	code, body = h.do(
		"PUT", "/api/settings", admin,
		map[string]any{"storageClasses": []string{"longhorn"}, "serverNotice": "  Maintenance at 22:00  "},
	)
	expect(t, "notice", code, 200, body)
	code, body = h.do("GET", "/api/settings", h.login("alice"), nil)
	if code != 200 || !strings.Contains(body, `"serverNotice":"Maintenance at 22:00"`) {
		t.Errorf("users must read the trimmed notice: %d %s", code, body)
	}
}

func TestLegalTextsArePublic(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do(
		"PUT",
		"/api/settings",
		admin,
		map[string]any{
			"storageClasses": []string{"longhorn"},
			"legalNotice":    "# Imprint\nMax Mustermann",
			"privacyPolicy":  strings.Repeat("x", 20001),
		},
	)
	expect(t, "too long privacy policy", code, 422, body)
	code, body = h.do(
		"PUT",
		"/api/settings",
		admin,
		map[string]any{
			"storageClasses": []string{"longhorn"},
			"legalNotice":    "# Imprint\nMax Mustermann",
			"privacyPolicy":  "We store usernames.",
		},
	)
	expect(t, "legal texts", code, 200, body)
	code, body = h.do("GET", "/api/legal", "", nil)
	if code != 200 || !strings.Contains(body, "Max Mustermann") || !strings.Contains(body, "We store usernames.") {
		t.Errorf("legal texts without sign-in: %d %s", code, body)
	}
	if strings.Contains(body, "longhorn") || strings.Contains(body, "storageClasses") {
		t.Errorf("the public endpoint must only return the legal texts: %s", body)
	}
}

func TestBrandingIsPublic(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("GET", "/api/branding", "", nil)
	if code != 200 || !strings.Contains(body, `"name":"Kubedactyl"`) ||
		!strings.Contains(body, `"tagline":"Game servers on Kubernetes"`) {
		t.Errorf("defaults without sign-in: %d %s", code, body)
	}
	png := "data:image/png;base64," +
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	for _, bad := range []map[string]any{
		{"brandLogo": "data:text/html;base64,PHNjcmlwdD4="},
		{"favicon": "https://example.com/favicon.ico"},
		{"brandLogo": "data:image/png;base64," + strings.Repeat("A", 200000)},
		{"brandName": strings.Repeat("x", 41)},
		{"sessionHours": 721},
		{"apiTokenMaxDays": -1},
	} {
		bad["storageClasses"] = []string{"longhorn"}
		code, body = h.do("PUT", "/api/settings", admin, bad)
		expect(t, fmt.Sprintf("invalid settings %v", bad), code, 422, body)
	}
	code, body = h.do("PUT", "/api/settings", admin, map[string]any{
		"storageClasses": []string{"longhorn"},
		"brandName":      " Acme Games ",
		"brandTagline":   "Our servers",
		"brandLogo":      png,
		"favicon":        png,
	})
	expect(t, "branding", code, 200, body)
	code, body = h.do("GET", "/api/branding", "", nil)
	if code != 200 || !strings.Contains(body, `"name":"Acme Games"`) ||
		!strings.Contains(body, `"logo":"data:image/png`) ||
		!strings.Contains(body, `"favicon":"data:image/png`) {
		t.Errorf("branding without sign-in: %d %s", code, body)
	}
	if strings.Contains(body, "longhorn") {
		t.Errorf("the public endpoint must only return the branding: %s", body)
	}
}

// Logo and favicon are sent again only after they changed: the browser revalidates with the ETag.
func TestBrandingRevalidates(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	get := func(etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/branding", nil)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		w := httptest.NewRecorder()
		h.router.ServeHTTP(w, req)
		return w
	}
	first := get("")
	etag := first.Header().Get("ETag")
	if first.Code != 200 || etag == "" || first.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("first answer: %d etag %q cache %q", first.Code, etag, first.Header().Get("Cache-Control"))
	}
	if w := get(etag); w.Code != http.StatusNotModified || w.Body.Len() != 0 {
		t.Errorf("unchanged branding: %d, %d bytes", w.Code, w.Body.Len())
	}
	code, body := h.do("PUT", "/api/settings", admin,
		map[string]any{"storageClasses": []string{"longhorn"}, "brandName": "Acme Games"})
	expect(t, "branding", code, 200, body)
	if w := get(etag); w.Code != 200 || !strings.Contains(w.Body.String(), "Acme Games") {
		t.Errorf("changed branding: %d %s", w.Code, w.Body.String())
	}
}

func TestSchedulesAPI(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	for _, r := range []struct{ method, path string }{
		{"GET", "/api/servers/bob-srv/schedules"}, {"PUT", "/api/servers/bob-srv/schedules"},
		{"POST", "/api/servers/bob-srv/schedules/x/run"},
	} {
		code, body := h.do(r.method, r.path, alice, map[string]any{"items": []any{}})
		expect(t, "foreign "+r.method+" "+r.path, code, 404, body)
	}
	daily := map[string]any{"name": "Daily restart", "cron": "0 4 * * *", "enabled": true, "tasks": []map[string]any{
		{"action": "command", "payload": "say Restart in 5 minutes"}, {"action": "restart", "delaySeconds": 300},
	}}
	code, body := h.do(
		"PUT",
		"/api/servers/alice-srv/schedules",
		alice,
		map[string]any{
			"items": []any{
				map[string]any{"name": "x", "cron": "nope", "tasks": []any{map[string]any{"action": "stop"}}},
			},
		},
	)
	expect(t, "invalid cron", code, 422, body)
	code, body = h.do("PUT", "/api/servers/alice-srv/schedules", alice, map[string]any{"items": []any{daily}})
	expect(t, "owner saves schedules", code, 200, body)
	code, body = h.do("GET", "/api/servers/alice-srv/schedules", alice, nil)
	if code != 200 || !strings.Contains(body, `"name":"Daily restart"`) || !strings.Contains(body, `"nextRunAt"`) ||
		!strings.Contains(body, `"timeZone":"UTC"`) {
		t.Errorf("list schedules: %d %s", code, body)
	}
	code, body = h.do("POST", "/api/servers/alice-srv/schedules/unknown/run", alice, nil)
	expect(t, "run unknown schedule", code, 404, body)
	gs := &v1alpha1.GameServer{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: tenancy.Namespace("alice"), Name: "alice-srv"}, gs)
	if len(gs.Spec.Schedules) != 1 || gs.Spec.Schedules[0].Tasks[1].DelaySeconds != 300 {
		t.Errorf("stored schedules: %+v", gs.Spec.Schedules)
	}
}

func TestSuspendAndBackups(t *testing.T) {
	h := newHarness(t)
	alice, admin := h.login("alice"), h.login("admin")

	code, body := h.do("POST", "/api/servers/alice-srv/suspend", alice, map[string]bool{"suspended": true})
	expect(t, "user suspends", code, 403, body)
	code, body = h.do("POST", "/api/servers/alice-srv/suspend", admin, map[string]bool{"suspended": true})
	expect(t, "admin suspends", code, 200, body)
	gs := &v1alpha1.GameServer{}
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: tenancy.Namespace("alice"), Name: "alice-srv"}, gs)
	if !gs.Spec.Suspended || gs.Spec.State != v1alpha1.PowerStopped {
		t.Errorf("suspend: %+v", gs.Spec)
	}

	code, body = h.do("GET", "/api/servers/alice-srv", alice, nil)
	expect(t, "owner still sees a suspended server", code, 200, body)
	for _, r := range []struct{ method, path string }{
		{"POST", "/api/servers/alice-srv/power"}, {"POST", "/api/servers/alice-srv/command"},
		{"PATCH", "/api/servers/alice-srv"}, {"GET", "/api/servers/alice-srv/files/session"},
		{"GET", "/api/servers/alice-srv/schedules"}, {"GET", "/api/servers/alice-srv/jobs"},
		{"POST", "/api/servers/alice-srv/backups"}, {"GET", "/api/servers/alice-srv/ws"},
	} {
		code, body = h.do(r.method, r.path, alice, map[string]string{"signal": "start", "command": "x"})
		expect(t, "suspended: "+r.method+" "+r.path, code, 403, body)
	}
	code, body = h.do("GET", "/api/servers/alice-srv/schedules", admin, nil)
	expect(t, "admin manages a suspended server", code, 200, body)
	code, body = h.do("POST", "/api/servers/alice-srv/power", admin, map[string]string{"signal": "start"})
	expect(t, "even admins cannot start a suspended server", code, 403, body)

	code, body = h.do("POST", "/api/servers/alice-srv/suspend", admin, map[string]bool{"suspended": false})
	expect(t, "unsuspend", code, 200, body)
	code, body = h.do("GET", "/api/servers/alice-srv/jobs", alice, nil)
	if code != 200 || !strings.Contains(body, `"items":[]`) {
		t.Errorf("jobs: %d %s", code, body)
	}
	code, body = h.do("POST", "/api/servers/bob-srv/backups", alice, nil)
	expect(t, "foreign backup", code, 404, body)
	code, body = h.do("POST", "/api/servers/alice-srv/backups/..%2Fetc.tar.gz/restore", alice, nil)
	expect(t, "invalid backup name", code, 404, body)

	// Restoring needs a stopped server.
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.Suspended, gs.Spec.State = false, v1alpha1.PowerRunning
	_ = h.client.Patch(t.Context(), gs, patch)
	code, body = h.do("POST", "/api/servers/alice-srv/backups/backup-2026-09-29_040005.tar.gz/restore", alice, nil)
	expect(t, "restore while running", code, 409, body)
}

func TestUpgradeDisabled(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	code, body := h.do("GET", "/api/upgrade", admin, nil)
	if code != 200 || !strings.Contains(body, `"enabled":false`) {
		t.Errorf("status without self-upgrades: %d %s", code, body)
	}
	code, body = h.do("POST", "/api/upgrade", admin, map[string]string{"version": "9.9.9"})
	expect(t, "upgrade without self-upgrades", code, 409, body)
}

// noDNS answers every lookup with "no such host", so tests never use the network.
type noDNS struct{}

func (noDNS) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return nil, errors.New("no such host")
}

func TestLogoutEndsTheSession(t *testing.T) {
	h := newHarness(t)
	token := h.login("alice")
	code, body := h.do("POST", "/api/auth/logout", token, nil)
	expect(t, "logout", code, 204, body)
	code, body = h.do("GET", "/api/auth/me", token, nil)
	expect(t, "token after logout", code, 401, body)
	other := h.login("alice")
	code, body = h.do("GET", "/api/auth/me", other, nil)
	expect(t, "other sessions stay", code, 200, body)
}

func TestConsoleChecksCredentialsAgain(t *testing.T) {
	h := newHarness(t)
	ctx := t.Context()
	alice := h.login("alice")
	s := &consoleSession{api: h.api, token: alice, namespace: tenancy.Namespace("alice"), server: "alice-srv"}
	if !s.authorized(ctx) {
		t.Fatal("the owner may use the console")
	}
	if foreign := (&consoleSession{api: h.api, token: alice, namespace: tenancy.Namespace("bob")}); foreign.authorized(
		ctx,
	) {
		t.Error("a console of another user's server")
	}
	h.do("POST", "/api/auth/logout", alice, nil)
	if s.authorized(ctx) {
		t.Error("an open console must end after the logout")
	}
	bob := h.login("bob")
	u := &v1alpha1.User{}
	_ = h.client.Get(ctx, client.ObjectKey{Namespace: sysNS, Name: "bob"}, u)
	u.Spec.Disabled = true
	_ = h.client.Update(ctx, u)
	if (&consoleSession{api: h.api, token: bob, namespace: tenancy.Namespace("bob")}).authorized(ctx) {
		t.Error("an open console must end when the account is disabled")
	}
}
