package httpapi

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/serverctl"
	"app/internal/tenancy"
)

// IndexServerName indexes game servers by name; names are unique across namespaces.
const IndexServerName = "metadata.name"

// ServerNameIndex extracts the index value (register it with the manager's field indexer).
func ServerNameIndex(o client.Object) []string { return []string{o.GetName()} }

var gameServerResource = schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "gameservers"}

// findServer looks a server up by name in all namespaces the caller may access.
// Servers of other users are reported as not found, so their existence is not revealed.
func (a *API) findServer(c *gin.Context, name string) (*v1alpha1.GameServer, error) {
	var list v1alpha1.GameServerList
	if err := a.Client.List(c, &list, client.MatchingFields{IndexServerName: name}); err != nil {
		return nil, err
	}
	p := principal(c)
	if gs := accessible(p, list.Items, name); gs != nil {
		return gs, nil
	}
	// Not in the informer cache yet: a server created a moment ago (its watch can lag by minutes).
	items, err := a.readServers(c, scopeOf(p))
	if err != nil {
		return nil, err
	}
	if gs := accessible(p, items, name); gs != nil {
		return gs, nil
	}
	return nil, apierrors.NewNotFound(gameServerResource, name)
}

// accessible returns the server with that name the caller may access, or nil.
func accessible(p *Principal, items []v1alpha1.GameServer, name string) *v1alpha1.GameServer {
	for i := range items {
		gs := &items[i]
		if gs.Name == name && tenancy.Owns(gs.Namespace) && (p.Admin() || gs.Namespace == p.Namespace) {
			return gs
		}
	}
	return nil
}

// serverListTTL is how long the server lists are kept (every viewer polls them every few seconds).
const serverListTTL = 2 * time.Second

// scopeOf is the key of the servers the caller sees: "" for every namespace (administrators), otherwise the
// user's namespace.
func scopeOf(p *Principal) string {
	if p.Admin() {
		return ""
	}
	return p.Namespace
}

// readServers lists the servers of a scope (see scopeOf) from the API server, not the informer cache: its watch
// can lag by minutes behind a proxy that cuts streams, so a deleted server stayed and a new one was missing. The
// list is read at most once per scope and serverListTTL; changes made through the panel drop it (forgetServers).
// The result is the caller's own copy.
func (a *API) readServers(ctx context.Context, scope string) ([]v1alpha1.GameServer, error) {
	items, err := a.serverLists.Get(scope, func() ([]v1alpha1.GameServer, error) {
		var list v1alpha1.GameServerList
		var opts []client.ListOption
		if scope != "" {
			opts = append(opts, client.InNamespace(scope))
		}
		if err := a.Reader.List(ctx, &list, opts...); err != nil {
			return nil, err
		}
		// A deleted server stays until its volume is released; it is gone for the list already.
		items := slices.DeleteFunc(ownServers(list.Items), func(gs v1alpha1.GameServer) bool {
			return gs.DeletionTimestamp != nil
		})
		return items, nil
	})
	return slices.Clone(items), err
}

// forgetServers drops the kept server lists after a change made through the panel (the administrators' list and
// those of the namespaces), so the next request shows it.
func (a *API) forgetServers(namespaces ...string) {
	a.serverLists.Forget("")
	for _, ns := range namespaces {
		a.serverLists.Forget(ns)
	}
}

// serverNameTaken reports whether a server name exists in any namespace (read uncached: a
// server created a moment ago must count).
func (a *API) serverNameTaken(ctx context.Context, name string) (bool, error) {
	var list v1alpha1.GameServerList
	if err := a.Reader.List(ctx, &list); err != nil {
		return false, err
	}
	return slices.ContainsFunc(list.Items, func(gs v1alpha1.GameServer) bool { return gs.Name == name }), nil
}

// ownServers drops servers that belong to another installation in the same cluster.
func ownServers(items []v1alpha1.GameServer) []v1alpha1.GameServer {
	out := items[:0]
	for _, gs := range items {
		if tenancy.Owns(gs.Namespace) {
			out = append(out, gs)
		}
	}
	return out
}

// reloadServer reads the current state of a server (e.g. for a long running websocket).
func (a *API) reloadServer(ctx context.Context, namespace, name string) (*v1alpha1.GameServer, bool) {
	gs := &v1alpha1.GameServer{}
	if err := a.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gs); err != nil {
		return nil, false
	}
	return gs, true
}

// visibleServer strips variables that the egg hides from users. With an external domain,
// users see the domain as the address instead of the load balancer IP.
func visibleServer(p *Principal, gs *v1alpha1.GameServer, e *v1alpha1.Egg, domain string) *v1alpha1.GameServer {
	if p.Admin() {
		return gs
	}
	out := gs.DeepCopy()
	if domain != "" {
		out.Spec.LoadBalancerIP = ""
		if out.Status.Address != "" {
			out.Status.Address = domain
		}
	}
	if e == nil {
		return out
	}
	viewable := map[string]bool{}
	for _, v := range e.Spec.Variables {
		viewable[v.EnvVariable] = v.UserViewable
	}
	for k := range out.Spec.Environment {
		if !viewable[k] {
			delete(out.Spec.Environment, k)
		}
	}
	return out
}

// visibleEgg strips hidden variables and the install script for users.
func visibleEgg(p *Principal, e *v1alpha1.Egg) *v1alpha1.Egg {
	if p.Admin() {
		return e
	}
	out := e.DeepCopy()
	out.Spec.Install = v1alpha1.InstallScript{}
	out.Spec.Variables = nil
	for _, v := range e.Spec.Variables {
		if v.UserViewable {
			out.Spec.Variables = append(out.Spec.Variables, v)
		}
	}
	return out
}

var errAdminOnly = errors.New("only administrators may change this setting")

// checkUserUpdate enforces what a non-admin may change on a server.
func checkUserUpdate(req *UpdateServerRequest, gs *v1alpha1.GameServer, e *v1alpha1.Egg) error {
	if req.Startup != nil || req.MemoryMiB != nil || req.CPUMillis != nil || req.DiskMiB != nil || req.Ports != nil ||
		req.LoadBalancerIP != nil || req.ExternalTrafficPolicy != nil || req.StopTimeoutSeconds != nil {
		return forbidden(errAdminOnly)
	}
	if req.Image != nil && *req.Image != gs.Spec.Image {
		allowed := false
		for _, img := range e.Spec.DockerImages {
			allowed = allowed || img.Image == *req.Image
		}
		if !allowed {
			return forbidden(errors.New("only the images of the egg can be selected"))
		}
	}
	editable := map[string]bool{}
	for _, v := range e.Spec.Variables {
		editable[v.EnvVariable] = v.UserEditable
	}
	for k, v := range req.Environment {
		if !editable[k] && gs.Spec.Environment[k] != v {
			return forbidden(errors.New("variable " + k + " cannot be changed"))
		}
	}
	return nil
}

// notSuspended blocks everything but viewing for owners of a suspended server.
func (a *API) notSuspended(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if gs.Spec.Suspended && !principal(c).Admin() {
		a.fail(c, serverctl.ErrSuspended)
		return
	}
	c.Set(serverKey, gs)
	c.Next()
}
