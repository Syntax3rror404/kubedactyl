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
	"app/internal/egg"
	"app/internal/gameserver"
	"app/internal/serverctl"
	"app/internal/tenancy"
	"app/internal/users"
)

// IndexServerName indexes game servers by name; names are unique across namespaces.
const IndexServerName = "metadata.name"

// ServerNameIndex extracts the index value (register it with the manager's field indexer).
func ServerNameIndex(o client.Object) []string { return []string{o.GetName()} }

var gameServerResource = schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "gameservers"}

// access is what the caller may do on a server.
type access int

const (
	accessNone access = iota
	// accessView: a subuser sees status, address and console output.
	accessView
	// accessControl: a subuser also starts and stops the server and sends console commands.
	accessControl
	// accessFull: the owner or an administrator.
	accessFull
)

// accessTo returns what the caller may do on a server. A subuser needs a share on a server of its owner: the
// owner check keeps a share from working after a transfer or for a later account with the same name.
func (p *Principal) accessTo(gs *v1alpha1.GameServer) access {
	switch {
	case p.Admin() || (p.Namespace != "" && gs.Namespace == p.Namespace):
		return accessFull
	case !p.Subuser() || gs.Namespace != p.home():
		return accessNone
	}
	for _, s := range gs.Spec.SharedWith {
		if s.User != p.User.Name {
			continue
		}
		if s.Access == v1alpha1.ShareControl {
			return accessControl
		}
		return accessView
	}
	return accessNone
}

// home is the namespace of the servers the caller can see: its own, for a subuser its owner's.
func (p *Principal) home() string { return users.Home(p.User) }

var errNoAccess = errors.New("the owner has not shared this with you")

// requireAccess refuses callers with less access to the server (subusers).
func (a *API) requireAccess(least access) gin.HandlerFunc {
	return func(c *gin.Context) {
		gs, ok := a.loadServer(c)
		if !ok {
			return
		}
		if principal(c).accessTo(gs) < least {
			a.fail(c, forbidden(errNoAccess))
			return
		}
		c.Set(serverKey, gs)
		c.Next()
	}
}

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
		if p.accessTo(gs) != accessNone {
			return gs, nil
		}
	}
	return nil, apierrors.NewNotFound(gameServerResource, name)
}

// serverNameTaken reports whether a server name exists in any namespace (read uncached: a
// server created a moment ago must count). The API server filters by name.
func (a *API) serverNameTaken(ctx context.Context, name string) (bool, error) {
	var list v1alpha1.GameServerList
	if err := a.Reader.List(ctx, &list, client.MatchingFields{"metadata.name": name}); err != nil {
		return false, err
	}
	return len(list.Items) > 0, nil
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

// hideFromUser strips what a user may not see from their server, in place (handlers pass their own
// copy): variables the egg hides and, with an external domain, the load balancer IPs (the domain is
// the address then; a ClusterIP server keeps its DNS name inside the cluster).
func hideFromUser(gs *v1alpha1.GameServer, e *v1alpha1.Egg, domain string) {
	if domain != "" && !gameserver.ClusterOnly(gs) {
		gs.Spec.LoadBalancerIP = ""
		if gs.Status.Address != "" {
			gs.Status.Address = domain
		}
		gs.Status.Addresses = nil
	}
	if e == nil {
		return
	}
	viewable := map[string]bool{}
	for _, v := range e.Spec.Variables {
		viewable[v.EnvVariable] = v.UserViewable
	}
	for k := range gs.Spec.Environment {
		if !viewable[k] {
			delete(gs.Spec.Environment, k)
		}
	}
}

// hideFromSubuser strips what only the owner sees from a server shared with a subuser, in place: variables,
// startup command, schedules and the other shares.
func hideFromSubuser(gs *v1alpha1.GameServer, name string) {
	gs.Spec.Environment, gs.Spec.Startup, gs.Spec.Schedules = nil, "", nil
	gs.Spec.SharedWith = slices.DeleteFunc(gs.Spec.SharedWith, func(s v1alpha1.ServerShare) bool {
		return s.User != name
	})
}

// visibleEgg fills in the startup commands of older eggs and, for users, strips hidden variables
// and the install script. It changes the egg in place (handlers pass their own copy).
func visibleEgg(p *Principal, e *v1alpha1.Egg) *v1alpha1.Egg {
	e.Spec.StartupCommands = egg.StartupCommands(&e.Spec)
	if p.Admin() {
		return e
	}
	e.Spec.Install = v1alpha1.InstallScript{}
	variables := e.Spec.Variables
	e.Spec.Variables = nil
	for _, v := range variables {
		if v.UserViewable {
			e.Spec.Variables = append(e.Spec.Variables, v)
		}
	}
	return e
}

var errAdminOnly = errors.New("only administrators may change this setting")

// checkUserUpdate enforces what a non-admin may change on a server.
func checkUserUpdate(req *UpdateServerRequest, gs *v1alpha1.GameServer, e *v1alpha1.Egg) error {
	if req.Startup != nil || req.MemoryMiB != nil || req.CPUMillis != nil || req.DiskMiB != nil || req.Ports != nil ||
		req.LoadBalancerIP != nil || req.ExternalTrafficPolicy != nil || req.IPv6 != nil ||
		req.StopTimeoutSeconds != nil {
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

// notLocked blocks everything but viewing for everyone while a server is being removed (its finalizer still
// runs) or its files move to another storage class, and for the owners of a suspended server.
func (a *API) notLocked(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if gs.DeletionTimestamp != nil {
		a.fail(c, serverctl.ErrRemoving)
		return
	}
	if gameserver.Migrating(gs) {
		a.fail(c, serverctl.ErrMigrating)
		return
	}
	if gs.Spec.Suspended && !principal(c).Admin() {
		a.fail(c, serverctl.ErrSuspended)
		return
	}
	c.Set(serverKey, gs)
	c.Next()
}
