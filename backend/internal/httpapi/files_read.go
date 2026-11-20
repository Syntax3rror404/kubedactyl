package httpapi

import (
	"io"
	"net/http"
	"path"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"app/api/v1alpha1"
	"app/base/api"
	"app/internal/files"
)

// maxEditableSize is the largest file the editor opens.
const maxEditableSize = 5 << 20

// FileList is a directory listing.
type FileList struct {
	Directory string        `json:"directory" example:"/"`
	Items     []files.Entry `json:"items"`
}

// denylist returns the egg's file_denylist; a server whose egg is gone has none. Any other
// failure answers the request (fail closed).
func (a *API) denylist(w http.ResponseWriter, r *http.Request, gs *v1alpha1.GameServer) ([]string, bool) {
	e, err := a.Eggs.Get(r.Context(), gs.Spec.EggRef)
	switch {
	case apierrors.IsNotFound(err):
		return nil, true
	case err != nil:
		a.Fail(w, r, err)
		return nil, false
	}
	return e.Spec.FileDenylist, true
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
func (a *API) listFiles(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	deny, ok := a.denylist(w, r, gs)
	if !ok {
		return
	}
	dir := cleanDir(r.URL.Query().Get("directory"))
	items, err := a.Files.Manager.List(r.Context(), files.RefOf(gs), dir, deny)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	if items == nil {
		items = []files.Entry{}
	}
	api.JSON(w, http.StatusOK, FileList{Directory: dir, Items: items})
}

// readFile godoc
//
//	@Summary	Read a file (max. 5 MiB)
//	@Tags		Files
//	@Produce	plain
//	@Param		server	path		string	true	"Server name"
//	@Param		file	query		string	true	"File path"
//	@Success	200		{string}	string
//	@Failure	413		{object}	api.ErrorResponse
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/contents [get]
func (a *API) readFile(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	deny, ok := a.denylist(w, r, gs)
	if !ok {
		return
	}
	data, err := a.Files.Manager.Read(r.Context(), files.RefOf(gs), r.URL.Query().Get("file"), maxEditableSize, deny)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "file read", "server", gs.Name, "file", r.URL.Query().Get("file"))
	api.Data(w, http.StatusOK, "text/plain; charset=utf-8", data)
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
func (a *API) downloadFile(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	deny, ok := a.denylist(w, r, gs)
	if !ok {
		return
	}
	file := r.URL.Query().Get("file")
	// The request context ends the copy in the files pod when the browser cancels the download.
	err := a.Download(w, r, path.Base(file), func(out io.Writer) error {
		return a.Files.Manager.Download(r.Context(), files.RefOf(gs), file, out, deny)
	})
	if err == nil {
		a.Audit(r, "file downloaded", "server", gs.Name, "file", file)
	}
}
