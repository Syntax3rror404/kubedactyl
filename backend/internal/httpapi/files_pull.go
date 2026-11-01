package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/files"
)

// PullFileRequest downloads a file from a URL into the server files.
type PullFileRequest struct {
	URL       string `json:"url"                 binding:"required" example:"https://example.com/plugin.jar"`
	Directory string `json:"directory,omitempty"                    example:"/plugins"`
	// Filename defaults to the last path segment of the URL.
	Filename string `json:"filename,omitempty"`
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
//	@Failure		422		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/pull [post]
func (a *API) pullFile(c *gin.Context) {
	var req PullFileRequest
	if !a.bindJSON(c, &req) {
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	deny, ok := a.denylist(c, gs)
	if !ok {
		return
	}
	job, err := a.Files.StartPull(files.RefOf(gs), req.URL, cleanDir(req.Directory), req.Filename, deny)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "download started", "server", gs.Name, "url", req.URL, "file", job.Label)
	c.JSON(http.StatusAccepted, job)
}
