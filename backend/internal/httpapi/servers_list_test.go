package httpapi

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/tenancy"
	"app/internal/testutil"
)

// The server list and the server lookup show what the API server has, also while the informer cache lags behind:
// a server created a moment ago is there, a deleted one is gone, also while its finalizer still runs.
func TestServerListIgnoresCacheLag(t *testing.T) {
	h := newHarness(t)
	alice := h.login("alice")
	admin := h.login("admin")

	// The API server: alice-srv was deleted, bob-srv is being deleted and fresh-srv was created; the cache
	// (h.client) has seen none of it.
	var cached v1alpha1.GameServerList
	if err := h.client.List(context.Background(), &cached); err != nil {
		t.Fatal(err)
	}
	// bob-srv is being deleted (its finalizer still runs).
	now := metav1.Now()
	for i := range cached.Items {
		if gs := &cached.Items[i]; gs.Name == "bob-srv" {
			gs.DeletionTimestamp, gs.Finalizers = &now, []string{"kubedactyl.io/volume"}
		}
	}
	b := testutil.Builder(t)
	for i := range cached.Items {
		if gs := &cached.Items[i]; gs.Name != "alice-srv" {
			gs.ResourceVersion = ""
			b = b.WithObjects(gs)
		}
	}
	fresh := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{
		Name: "fresh-srv", Namespace: tenancy.Namespace("alice"),
		Labels: map[string]string{tenancy.LabelOwner: "alice"},
	}, Spec: v1alpha1.GameServerSpec{EggRef: "paper"}}
	h.api.Reader = b.WithObjects(fresh).Build()

	code, body := h.do("GET", "/api/servers", alice, nil)
	if code != 200 || !strings.Contains(body, `"fresh-srv"`) || strings.Contains(body, `"alice-srv"`) {
		t.Errorf("alice's list: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/servers", admin, nil)
	if code != 200 || !strings.Contains(body, `"fresh-srv"`) || strings.Contains(body, `"bob-srv"`) ||
		strings.Contains(body, `"alice-srv"`) || strings.Contains(body, "foreign-srv") {
		t.Errorf("admin's list: %d %s", code, body)
	}
	code, body = h.do("GET", "/api/servers/fresh-srv", alice, nil)
	expect(t, "server missing in the cache", code, 200, body)
	code, body = h.do("GET", "/api/servers/fresh-srv", h.login("bob"), nil)
	expect(t, "foreign server missing in the cache", code, 404, body)
}
