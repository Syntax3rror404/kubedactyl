package httpapi

import (
	"mime"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/files"
)

// maxEditableSize is the largest file the editor opens.
const maxEditableSize = 5 << 20

// FileList is a directory listing.
type FileList struct {
	Directory string        `json:"directory" example:"/"`
	Items     []files.Entry `json:"items"`
}

// denylist returns the egg's file_denylist.
func (a *API) denylist(c *gin.Context, gs *v1alpha1.GameServer) []string {
	e, err := a.Eggs.Get(c, gs.Spec.EggRef)
	if err != nil {
		return nil
	}
	return e.Spec.FileDenylist
}

// cleanDir turns a directory from a request into an absolute, clean path below the server root.
func cleanDir(dir string) string { return path.Clean("/" + dir) }

// inRoot returns the path of an entry inside the directory root.
func inRoot(root, name string) string { return path.Join("/", root, name) }

// listFiles godoc
//
//	@Summary	List a directory
//	@Tags		Files
//	@Produce	json
//	@Param		server		path		string	true	"Server name"
//	@Param		directory	query		string	false	"Directory relative to the server root"	default(/)
//	@Success	200			{object}	FileList
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/list [get]
func (a *API) listFiles(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	dir := cleanDir(c.Query("directory"))
	items, err := a.Files.Manager.List(c, files.RefOf(gs), dir, a.denylist(c, gs))
	if err != nil {
		a.fail(c, err)
		return
	}
	if items == nil {
		items = []files.Entry{}
	}
	c.JSON(http.StatusOK, FileList{Directory: dir, Items: items})
}

// readFile godoc
//
//	@Summary	Read a file (max. 5 MiB)
//	@Tags		Files
//	@Produce	plain
//	@Param		server	path		string	true	"Server name"
//	@Param		file	query		string	true	"File path"
//	@Success	200		{string}	string
//	@Failure	413		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/contents [get]
func (a *API) readFile(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	data, err := a.Files.Manager.Read(c, files.RefOf(gs), c.Query("file"), maxEditableSize, a.denylist(c, gs))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "file read", "server", gs.Name, "file", c.Query("file"))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}

// downloadFile godoc
//
//	@Summary	Download a file
//	@Tags		Files
//	@Produce	octet-stream
//	@Param		server	path	string	true	"Server name"
//	@Param		file	query	string	true	"File path"
//	@Success	200
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/download [get]
func (a *API) downloadFile(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	file := c.Query("file")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": path.Base(file)}))
	c.Status(http.StatusOK)
	err := a.Files.Manager.Download(c, files.RefOf(gs), file, c.Writer, a.denylist(c, gs))
	switch {
	case err == nil:
		a.audit(c, "file downloaded", "server", gs.Name, "file", file)
	case !c.Writer.Written():
		a.fail(c, err)
	}
}
