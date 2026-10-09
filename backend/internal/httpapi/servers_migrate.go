package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/settings"
)

// MigrateRequest names the storage class the server files move to.
type MigrateRequest struct {
	StorageClass string `json:"storageClass" binding:"required" example:"longhorn"`
	// StartOnCompletion starts the server once the migration has ended.
	StartOnCompletion bool `json:"startOnCompletion,omitempty" example:"true"`
}

// migrateServer godoc
//
//	@Summary		Move a server to another storage class
//	@Description	Stops the server and copies its files, backups included, to a new volume of the storage class
//	@Description	(status.migration). The server is locked until it uses the checked copy; the old volume is
//	@Description	deleted then. A failed copy leaves the server on its volume. With startOnCompletion the server
//	@Description	starts when the migration has ended.
//	@Tags			Servers
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string			true	"Server name"
//	@Param			body	body		MigrateRequest	true	"Storage class"
//	@Success		200		{object}	v1alpha1.GameServer
//	@Failure		409		{object}	ErrorResponse	"a job or the install runs, or it is the current class"
//	@Failure		422		{object}	ErrorResponse	"the storage class is not enabled"
//	@Security		BearerAuth
//	@Router			/servers/{server}/migrate [post]
func (a *API) migrateServer(c *gin.Context) {
	var req MigrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	set, err := a.Settings.Current(c)
	if err == nil {
		_, err = settings.Choose(req.StorageClass, "", set.StorageClasses, "storage class")
	}
	if err == nil {
		err = a.Ops.Migrate(c, gs, req.StorageClass, req.StartOnCompletion)
	}
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "server storage migration started", "server", gs.Name, "from", gs.Status.StorageClass,
		"to", req.StorageClass)
	a.Trigger(gs.Namespace, gs.Name)
	c.JSON(http.StatusOK, gs)
}

// cancelMigration godoc
//
//	@Summary		Cancel a storage migration
//	@Description	Ends a migration before the server switches to the new volume; the copy is deleted and the server
//	@Description	stays on its volume.
//	@Tags			Servers
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Success		200		{object}	v1alpha1.GameServer
//	@Failure		409		{object}	ErrorResponse	"no migration is running or it is switching already"
//	@Security		BearerAuth
//	@Router			/servers/{server}/migrate/cancel [post]
func (a *API) cancelMigration(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Ops.CancelMigration(c, gs); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "server storage migration cancelled", "server", gs.Name)
	a.Trigger(gs.Namespace, gs.Name)
	c.JSON(http.StatusOK, gs)
}
