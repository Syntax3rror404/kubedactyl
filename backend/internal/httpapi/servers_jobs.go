package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/files"
)

// JobList lists the recent background jobs of a server (backups, restores, downloads).
type JobList struct {
	Items []files.Job `json:"items"`
}

// listJobs godoc
//
//	@Summary	Background jobs of a server
//	@Tags		Backups
//	@Produce	json
//	@Param		server	path		string	true	"Server name"
//	@Success	200		{object}	JobList
//	@Security	BearerAuth
//	@Router		/servers/{server}/jobs [get]
func (a *API) listJobs(c *gin.Context) {
	if gs, ok := a.loadServer(c); ok {
		c.JSON(http.StatusOK, JobList{Items: a.Files.Jobs(files.RefOf(gs))})
	}
}
