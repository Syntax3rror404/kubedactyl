package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PowerRequest changes the power state.
type PowerRequest struct {
	Signal string `json:"signal" binding:"required,oneof=start stop restart kill" enums:"start,stop,restart,kill" example:"start"`
}

// CommandRequest sends a console command.
type CommandRequest struct {
	Command string `json:"command" binding:"required" example:"say Hello"`
}

// SuspendRequest suspends or unsuspends a server.
type SuspendRequest struct {
	Suspended bool `json:"suspended"`
}

// sendPower godoc
//
//	@Summary		Change the power state
//	@Description	start, stop (egg stop command or signal, killed after the stop timeout), restart, kill (immediate
//	@Description	SIGKILL).
//	@Tags			Servers
//	@Accept			json
//	@Param			server	path	string			true	"Server name"
//	@Param			body	body	PowerRequest	true	"Power action"
//	@Success		204
//	@Security		BearerAuth
//	@Router			/servers/{server}/power [post]
func (a *API) sendPower(c *gin.Context) {
	var req PowerRequest
	if !a.bindJSON(c, &req) {
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Ops.Power(c, gs, req.Signal); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "power action sent", "server", gs.Name, "signal", req.Signal)
	a.Trigger(gs.Namespace, gs.Name)
	c.Status(http.StatusNoContent)
}

// sendCommand godoc
//
//	@Summary	Send a console command
//	@Tags		Servers
//	@Accept		json
//	@Param		server	path	string			true	"Server name"
//	@Param		body	body	CommandRequest	true	"Command"
//	@Success	204
//	@Failure	409	{object}	ErrorResponse	"server is not running"
//	@Security	BearerAuth
//	@Router		/servers/{server}/command [post]
func (a *API) sendCommand(c *gin.Context) {
	var req CommandRequest
	if !a.bindJSON(c, &req) {
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Ops.Command(c, gs, req.Command); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "console command sent", "server", gs.Name, "command", req.Command)
	c.Status(http.StatusNoContent)
}

// reinstallServer godoc
//
//	@Summary		Reinstall a game server
//	@Description	Stops the server and runs the egg install script again. Files are kept unless the script removes
//	@Description	them.
//	@Tags			Servers
//	@Param			server	path	string	true	"Server name"
//	@Success		204
//	@Security		BearerAuth
//	@Router			/servers/{server}/reinstall [post]
func (a *API) reinstallServer(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Ops.Reinstall(c, gs); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "server reinstall started", "server", gs.Name)
	a.Trigger(gs.Namespace, gs.Name)
	c.Status(http.StatusNoContent)
}

// suspendServer godoc
//
//	@Summary		Suspend or unsuspend a server
//	@Description	A suspended server is stopped; its owner can only view it until an administrator unsuspends it.
//	@Tags			Servers
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string			true	"Server name"
//	@Param			body	body		SuspendRequest	true	"State"
//	@Success		200		{object}	v1alpha1.GameServer
//	@Security		BearerAuth
//	@Router			/servers/{server}/suspend [post]
func (a *API) suspendServer(c *gin.Context) {
	var req SuspendRequest
	if !a.bindJSON(c, &req) {
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Ops.Suspend(c, gs, req.Suspended); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "server suspension changed", "server", gs.Name, "suspended", req.Suspended)
	a.Trigger(gs.Namespace, gs.Name)
	c.JSON(http.StatusOK, gs)
}
