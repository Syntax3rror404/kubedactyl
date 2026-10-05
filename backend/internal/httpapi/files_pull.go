package httpapi

import (
	"context"
	"net/http"
	"path"

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
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	name, err := files.PullName(req.URL, req.Filename)
	if err != nil {
		a.fail(c, err)
		return
	}
	deny := a.denylist(c, gs)
	dir := cleanDir(req.Directory)
	if _, err := files.Resolve(path.Join(dir, name), deny); err != nil {
		a.fail(c, err)
		return
	}
	ref := files.RefOf(gs)
	job, err := a.Files.Start(ref, files.JobPull, path.Join(dir, name), func(ctx context.Context) error {
		return a.Files.Manager.Pull(ctx, ref, req.URL, dir, name, deny)
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "download started", "server", gs.Name, "url", req.URL, "file", path.Join(dir, name))
	c.JSON(http.StatusAccepted, job)
}
