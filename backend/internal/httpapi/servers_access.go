package httpapi

import (
	"context"
	"errors"
	"slices"

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
	for i := range list.Items {
		gs := &list.Items[i]
		if !tenancy.Owns(gs.Namespace) {
			continue
		}
		if p.Admin() || gs.Namespace == p.Namespace {
			return gs, nil
		}
	}
	return nil, apierrors.NewNotFound(gameServerResource, name)
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
