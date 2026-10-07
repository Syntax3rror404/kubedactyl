package httpapi

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
)

// GameServerList is a list of game servers.
type GameServerList struct {
	Items []v1alpha1.GameServer `json:"items"`
}

// serverKey holds the server that notSuspended or requireFilesPod already looked up.
const serverKey = "kubedactyl.server"

func (a *API) loadServer(c *gin.Context) (*v1alpha1.GameServer, bool) {
	// Set by notSuspended or requireFilesPod, which already looked the server up.
	if v, ok := c.Get(serverKey); ok {
		return v.(*v1alpha1.GameServer), true
	}
	gs, err := a.findServer(c, c.Param("server"))
	if err != nil {
		a.fail(c, err)
		return nil, false
	}
	return gs, true
}

func (a *API) loadEgg(c *gin.Context, name string) (*v1alpha1.Egg, bool) {
	e, err := a.Eggs.Get(c, name)
	if err != nil {
		a.fail(c, err)
		return nil, false
	}
	return e, true
}

// listServers godoc
//
//	@Summary	List game servers
//	@Tags		Servers
//	@Produce	json
//	@Success	200	{object}	GameServerList
//	@Security	BearerAuth
//	@Router		/servers [get]
func (a *API) listServers(c *gin.Context) {
	p := principal(c)
	items, err := a.readServers(c, scopeOf(p))
	if err != nil {
		a.fail(c, err)
		return
	}
	list := v1alpha1.GameServerList{Items: items}
	eggs := a.eggsByName(c)
	domain := a.externalDomain(c)
	for i := range list.Items {
		list.Items[i] = *visibleServer(p, &list.Items[i], eggs[list.Items[i].Spec.EggRef], domain)
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return strings.ToLower(list.Items[i].Spec.DisplayName) < strings.ToLower(list.Items[j].Spec.DisplayName)
	})
	c.JSON(http.StatusOK, GameServerList{Items: list.Items})
}

// getServer godoc
//
//	@Summary	Get a game server
//	@Tags		Servers
//	@Produce	json
//	@Param		server	path		string	true	"Server name"
//	@Success	200		{object}	v1alpha1.GameServer
//	@Failure	404		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/servers/{server} [get]
func (a *API) getServer(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	var e *v1alpha1.Egg // stays nil when the egg is gone (like in listServers)
	if found, err := a.Eggs.Get(c, gs.Spec.EggRef); err == nil {
		e = found
	}
	c.JSON(http.StatusOK, visibleServer(principal(c), gs, e, a.externalDomain(c)))
}

// externalDomain is shown to users instead of the load balancer IP (read on every poll, so from
// the settings cache).
func (a *API) externalDomain(c *gin.Context) string {
	set, err := a.Settings.Current(c)
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
func (a *API) deleteServer(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Client.Delete(c, gs); err != nil {
		a.fail(c, err)
		return
	}
	a.forgetServers(gs.Namespace)
	a.audit(c, "server deleted", "server", gs.Name)
	c.Status(http.StatusNoContent)
}
