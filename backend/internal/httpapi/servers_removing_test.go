package httpapi

import (
	"context"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/tenancy"
)

// A server being removed (its finalizer still runs) stays visible, but nothing can be done with it anymore.
func TestRemovingServerIsLocked(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	gs, key := &v1alpha1.GameServer{}, client.ObjectKey{Namespace: tenancy.Namespace("bob"), Name: "bob-srv"}
	if err := h.client.Get(ctx, key, gs); err != nil {
		t.Fatal(err)
	}
	gs.Finalizers = []string{"kubedactyl.io/volume"}
	if err := h.client.Update(ctx, gs); err != nil {
		t.Fatal(err)
	}
	if err := h.client.Delete(ctx, gs); err != nil {
		t.Fatal(err)
	}
	admin, bob := h.login("admin"), h.login("bob")

	code, body := h.do("GET", "/api/servers", bob, nil)
	if code != 200 || !strings.Contains(body, `"deletionTimestamp"`) {
		t.Errorf("bob's list: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/servers/bob-srv", bob, nil)
	expect(t, "view", code, 200, body)
	for _, r := range []struct {
		token, path string
		body        any
	}{
		{bob, "/power", PowerRequest{Signal: "start"}},
		{admin, "/power", PowerRequest{Signal: "start"}},
		{admin, "/files/session", nil},
		{admin, "/suspend", SuspendRequest{Suspended: true}},
		{admin, "/transfer", TransferRequest{Owner: "alice"}},
	} {
		code, body := h.do("POST", "/api/servers/bob-srv"+r.path, r.token, r.body)
		expect(t, r.path, code, 409, body)
	}
}
