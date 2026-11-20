package httpapi

import (
	"net/http"

	"app/api/v1alpha1"
	"app/base/api"
	"app/base/validation"
	"app/internal/serverctl"
	"app/internal/tenancy"
)

// TransferRequest names the user a server moves to.
type TransferRequest struct {
	Owner string `json:"owner" example:"alice"`
}

// Validate checks that the new owner is named.
func (req TransferRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("owner", req.Owner != "")
	return errs.OrNil()
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
//	@Failure		409		{object}	api.ErrorResponse	"the server is running or already belongs to the user"
//	@Security		BearerAuth
//	@Router			/servers/{server}/transfer [post]
func (a *API) transferServer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if !a.Decode(w, r, &req) {
		return
	}
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	// Checked before ownerNamespace creates the target namespace, so a refused transfer leaves nothing behind.
	if err := checkTransfer(gs, req.Owner); err != nil {
		a.Fail(w, r, err)
		return
	}
	namespace, err := a.ownerNamespace(r.Context(), req.Owner)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	moved, err := a.Ops.Transfer(r.Context(), gs, req.Owner, namespace)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "server transferred", "server", gs.Name, "from", gs.Namespace, "to", namespace)
	a.Trigger(moved.Namespace, moved.Name)
	api.JSON(w, http.StatusOK, moved)
}

// checkTransfer refuses a server that already belongs to the owner or is not stopped.
func checkTransfer(gs *v1alpha1.GameServer, owner string) error {
	if gs.Namespace == tenancy.Namespace(owner) {
		return serverctl.ErrSameOwner
	}
	return serverctl.Stopped(gs)
}
