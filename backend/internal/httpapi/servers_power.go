package httpapi

import (
	"net/http"

	"app/base/api"
	"app/base/validation"
)

// PowerRequest changes the power state.
type PowerRequest struct {
	Signal string `json:"signal" enums:"start,stop,restart,kill" example:"start"`
}

// Validate checks that the signal is one of the four.
func (req PowerRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("signal", req.Signal != "")
	errs.OneOf("signal", req.Signal, "start", "stop", "restart", "kill")
	return errs.OrNil()
}

// CommandRequest sends a console command.
type CommandRequest struct {
	Command string `json:"command" example:"say Hello"`
}

// Validate checks that a command is given.
func (req CommandRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("command", req.Command != "")
	return errs.OrNil()
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
func (a *API) sendPower(w http.ResponseWriter, r *http.Request) {
	var req PowerRequest
	if !a.Decode(w, r, &req) {
		return
	}
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	if err := a.Ops.Power(r.Context(), gs, req.Signal); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "power action sent", "server", gs.Name, "signal", req.Signal)
	a.Trigger(gs.Namespace, gs.Name)
	w.WriteHeader(http.StatusNoContent)
}

// sendCommand godoc
//
//	@Summary	Send a console command
//	@Tags		Servers
//	@Accept		json
//	@Param		server	path	string			true	"Server name"
//	@Param		body	body	CommandRequest	true	"Command"
//	@Success	204
//	@Failure	409	{object}	api.ErrorResponse	"server is not running"
//	@Security	BearerAuth
//	@Router		/servers/{server}/command [post]
func (a *API) sendCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if !a.Decode(w, r, &req) {
		return
	}
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	if err := a.Ops.Command(r.Context(), gs, req.Command); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "console command sent", "server", gs.Name, "command", req.Command)
	w.WriteHeader(http.StatusNoContent)
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
func (a *API) reinstallServer(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	if err := a.Ops.Reinstall(r.Context(), gs); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "server reinstall started", "server", gs.Name)
	a.Trigger(gs.Namespace, gs.Name)
	w.WriteHeader(http.StatusNoContent)
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
func (a *API) suspendServer(w http.ResponseWriter, r *http.Request) {
	var req SuspendRequest
	if !a.Decode(w, r, &req) {
		return
	}
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	if err := a.Ops.Suspend(r.Context(), gs, req.Suspended); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "server suspension changed", "server", gs.Name, "suspended", req.Suspended)
	a.Trigger(gs.Namespace, gs.Name)
	api.JSON(w, http.StatusOK, gs)
}
