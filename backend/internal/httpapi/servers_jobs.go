package httpapi

import (
	"net/http"

	"app/base/api"
	"app/internal/files"
)

// JobList lists the recent background jobs of a server (backups, restores, downloads, archives).
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
func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	if gs, ok := a.loadServer(w, r); ok {
		api.JSON(w, http.StatusOK, JobList{Items: a.Files.Jobs(files.RefOf(gs))})
	}
}

// cancelJob godoc
//
//	@Summary		Cancel a background job
//	@Description	Stops a running job; what it left half done is removed (a cancelled restore leaves the
//	@Description	server files incomplete). A finished job stays as it is.
//	@Tags			Backups
//	@Param			server	path	string	true	"Server name"
//	@Param			job		path	string	true	"Job ID"
//	@Success		204
//	@Security		BearerAuth
//	@Router			/servers/{server}/jobs/{job}/cancel [post]
func (a *API) cancelJob(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	if err := a.Files.Cancel(r.Context(), files.RefOf(gs), r.PathValue("job")); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "job cancelled", "server", gs.Name, "job", r.PathValue("job"))
	w.WriteHeader(http.StatusNoContent)
}
