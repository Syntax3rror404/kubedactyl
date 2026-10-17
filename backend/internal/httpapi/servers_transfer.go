package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TransferRequest names the user a server moves to.
type TransferRequest struct {
	Owner string `json:"owner" binding:"required" example:"alice"`
}

// transferServer godoc
//
//	@Summary		Transfer a server to another user
//	@Description	Moves a stopped server with its data into the namespace of another user. Name, settings, files and
//	@Description	backups stay; the address is fixed to the current one.
//	@Tags			Servers
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string			true	"Server name"
//	@Param			body	body		TransferRequest	true	"New owner"
//	@Success		200		{object}	v1alpha1.GameServer
//	@Failure		409		{object}	ErrorResponse	"the server is running or already belongs to the user"
//	@Security		BearerAuth
//	@Router			/servers/{server}/transfer [post]
func (a *API) transferServer(c *gin.Context) {
	var req TransferRequest
	if !a.bindJSON(c, &req) {
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	namespace, err := a.ownerNamespace(c, req.Owner)
	if err != nil {
		a.fail(c, err)
		return
	}
	// The move must finish even when the client (or a proxy) gives up waiting.
	moved, err := a.Ops.Transfer(context.WithoutCancel(c), gs, req.Owner, namespace)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "server transferred", "server", gs.Name, "from", gs.Namespace, "to", namespace)
	a.Trigger(moved.Namespace, moved.Name)
	c.JSON(http.StatusOK, moved)
}
