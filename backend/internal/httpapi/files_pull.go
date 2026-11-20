package httpapi

import (
	"net/http"

	"app/base/api"
	"app/base/validation"
	"app/internal/files"
)

// PullFileRequest downloads a file from a URL into the server files.
type PullFileRequest struct {
	URL       string `json:"url"                 example:"https://example.com/plugin.jar"`
	Directory string `json:"directory,omitempty" example:"/plugins"`
	// Filename defaults to the last path segment of the URL.
	Filename string `json:"filename,omitempty"`
}

// Validate checks that a URL is given.
func (req PullFileRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("url", req.URL != "")
	return errs.OrNil()
}

// pullFile godoc
//
//	@Summary		Download a file from a URL
//	@Description	The file container downloads the URL (http/https) into the directory in the background; the
//	@Description	network isolation of the server applies. Existing files are not overwritten.
//	@Tags			Files
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string			true	"Server name"
//	@Param			body	body		PullFileRequest	true	"URL"
//	@Success		202		{object}	files.Job
//	@Failure		422		{object}	api.ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/pull [post]
func (a *API) pullFile(w http.ResponseWriter, r *http.Request) {
	var req PullFileRequest
	if !a.Decode(w, r, &req) {
		return
	}
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	deny, ok := a.denylist(w, r, gs)
	if !ok {
		return
	}
	job, err := a.Files.StartPull(files.RefOf(gs), req.URL, cleanDir(req.Directory), req.Filename, deny)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "download started", "server", gs.Name, "url", req.URL, "file", job.Label)
	api.JSON(w, http.StatusAccepted, job)
}
