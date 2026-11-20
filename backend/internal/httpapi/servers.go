package httpapi

import (
	"context"
	"net/http"
	"slices"
	"sort"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/base/api"
)

// GameServerList is a list of game servers.
type GameServerList struct {
	Items []v1alpha1.GameServer `json:"items"`
}

// serverKey holds the server that the middleware notLocked, requireAccess or requireFilesPod already looked up.
type serverKey struct{}

// withServer passes the server a middleware looked up on to the handlers after it.
func withServer(r *http.Request, gs *v1alpha1.GameServer) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), serverKey{}, gs))
}

// serverOf is the server a middleware looked up (withServer).
func serverOf(r *http.Request) (*v1alpha1.GameServer, bool) {
	gs, ok := r.Context().Value(serverKey{}).(*v1alpha1.GameServer)
	return gs, ok
}

// loadServer returns the server of the path that the caller may access (findServer).
func (a *API) loadServer(w http.ResponseWriter, r *http.Request) (*v1alpha1.GameServer, bool) {
	if gs, ok := serverOf(r); ok {
		return gs, true
	}
	gs, err := a.findServer(r, r.PathValue("server"))
	if err != nil {
		a.Fail(w, r, err)
		return nil, false
	}
	return gs, true
}

// listServers godoc
//
//	@Summary	List game servers
//	@Tags		Servers
//	@Produce	json
//	@Success	200	{object}	GameServerList
//	@Security	BearerAuth
//	@Router		/servers [get]
func (a *API) listServers(w http.ResponseWriter, r *http.Request) {
	var list v1alpha1.GameServerList
	p := api.PrincipalOf(r)
	var opts []client.ListOption
	if !p.Admin() {
		opts = append(opts, client.InNamespace(home(p)))
	}
	if err := a.Client.List(r.Context(), &list, opts...); err != nil {
		a.Fail(w, r, err)
		return
	}
	list.Items = slices.DeleteFunc(list.Items, func(gs v1alpha1.GameServer) bool {
		return accessTo(p, &gs) == accessNone
	})
	eggs := a.eggsByName(w, r)
	domain := a.externalDomain(w, r)
	for i := range list.Items {
		visibleServer(p, &list.Items[i], eggs[list.Items[i].Spec.EggRef], domain)
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return strings.ToLower(list.Items[i].Spec.DisplayName) < strings.ToLower(list.Items[j].Spec.DisplayName)
	})
	api.JSON(w, http.StatusOK, GameServerList{Items: list.Items})
}

// getServer godoc
//
//	@Summary	Get a game server
//	@Tags		Servers
//	@Produce	json
//	@Param		server	path		string	true	"Server name"
//	@Success	200		{object}	v1alpha1.GameServer
//	@Failure	404		{object}	api.ErrorResponse
//	@Security	BearerAuth
//	@Router		/servers/{server} [get]
func (a *API) getServer(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	var e *v1alpha1.Egg // stays nil when the egg cannot be read: users then see no variables
	if found, err := a.Eggs.Get(r.Context(), gs.Spec.EggRef); err == nil {
		e = found
	}
	visibleServer(api.PrincipalOf(r), gs, e, a.externalDomain(w, r))
	api.JSON(w, http.StatusOK, gs)
}

// externalDomain is shown to users instead of the load balancer IP (read on every poll, so from
// the settings cache).
func (a *API) externalDomain(w http.ResponseWriter, r *http.Request) string {
	set, err := a.Settings.Current(r.Context())
	if err != nil {
		a.Log.Warn("reading panel settings", "err", err)
		return ""
	}
	return set.ExternalDomain
}

// deleteServer godoc
//
//	@Summary		Delete a game server
//	@Description	Deletes the server including its data volume.
//	@Tags			Servers
//	@Param			server	path	string	true	"Server name"
//	@Success		204
//	@Security		BearerAuth
//	@Router			/servers/{server} [delete]
func (a *API) deleteServer(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	if err := a.Client.Delete(r.Context(), gs); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "server deleted", "server", gs.Name)
	w.WriteHeader(http.StatusNoContent)
}
