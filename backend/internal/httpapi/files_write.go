package httpapi

import (
	"io"
	"net/http"
	"path"

	"app/base/api"
	"app/base/validation"
	"app/internal/files"
)

// CreateFolderRequest creates a directory.
type CreateFolderRequest struct {
	Root string `json:"root,omitempty" example:"/"`
	Name string `json:"name"           example:"plugins"`
}

// Validate checks that the folder has a name.
func (req CreateFolderRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("name", req.Name != "")
	return errs.OrNil()
}

// FilesRequest names entries inside a directory (delete, compress).
type FilesRequest struct {
	Root  string   `json:"root,omitempty" example:"/"`
	Files []string `json:"files"          example:"world,logs"`
}

// Validate checks that files are named.
func (req FilesRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("files", len(req.Files) > 0)
	return errs.OrNil()
}

// RenameFileRequest moves a file or directory.
type RenameFileRequest struct {
	Root string `json:"root,omitempty" example:"/"`
	From string `json:"from"           example:"old.txt"`
	To   string `json:"to"             example:"new.txt"`
}

// Validate checks that both names are given.
func (req RenameFileRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("from", req.From != "")
	errs.Require("to", req.To != "")
	return errs.OrNil()
}

// DecompressFileRequest extracts an archive.
type DecompressFileRequest struct {
	Root string `json:"root,omitempty" example:"/"`
	File string `json:"file"           example:"world.zip"`
}

// Validate checks that the archive is named.
func (req DecompressFileRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("file", req.File != "")
	return errs.OrNil()
}

// writeFile godoc
//
//	@Summary	Write a file
//	@Tags		Files
//	@Accept		plain
//	@Param		server	path	string	true	"Server name"
//	@Param		file	query	string	true	"File path"
//	@Param		body	body	string	true	"File content"
//	@Success	204
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/write [post]
func (a *API) writeFile(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	deny, ok := a.denylist(w, r, gs)
	if !ok {
		return
	}
	err := a.Files.Manager.Write(r.Context(), files.RefOf(gs), r.URL.Query().Get("file"), r.Body, deny)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "file written", "server", gs.Name, "file", r.URL.Query().Get("file"))
	w.WriteHeader(http.StatusNoContent)
}

// createFolder godoc
//
//	@Summary	Create a folder
//	@Tags		Files
//	@Accept		json
//	@Param		server	path	string				true	"Server name"
//	@Param		body	body	CreateFolderRequest	true	"Folder"
//	@Success	204
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/create-folder [post]
func (a *API) createFolder(w http.ResponseWriter, r *http.Request) {
	var req CreateFolderRequest
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
	folder := inRoot(req.Root, req.Name)
	if err := a.Files.Manager.CreateFolder(r.Context(), files.RefOf(gs), folder, deny); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "folder created", "server", gs.Name, "folder", folder)
	w.WriteHeader(http.StatusNoContent)
}

// deleteFiles godoc
//
//	@Summary	Delete files and folders
//	@Tags		Files
//	@Accept		json
//	@Param		server	path	string			true	"Server name"
//	@Param		body	body	FilesRequest	true	"Entries"
//	@Success	204
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/delete [post]
func (a *API) deleteFiles(w http.ResponseWriter, r *http.Request) {
	var req FilesRequest
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
	dir := cleanDir(req.Root)
	if err := a.Files.Manager.Delete(r.Context(), files.RefOf(gs), dir, req.Files, deny); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "files deleted", "server", gs.Name, "folder", dir, "files", req.Files)
	w.WriteHeader(http.StatusNoContent)
}

// renameFile godoc
//
//	@Summary		Rename or move a file
//	@Description	"to" may contain slashes to move the entry into another folder.
//	@Tags			Files
//	@Accept			json
//	@Param			server	path	string				true	"Server name"
//	@Param			body	body	RenameFileRequest	true	"Rename"
//	@Success		204
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/rename [put]
func (a *API) renameFile(w http.ResponseWriter, r *http.Request) {
	var req RenameFileRequest
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
	from := inRoot(req.Root, req.From)
	to := inRoot(req.Root, req.To)
	if err := a.Files.Manager.Rename(r.Context(), files.RefOf(gs), from, to, deny); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "file renamed", "server", gs.Name, "from", from, "to", to)
	w.WriteHeader(http.StatusNoContent)
}

// uploadFiles godoc
//
//	@Summary	Upload files
//	@Tags		Files
//	@Accept		mpfd
//	@Param		server		path		string	true	"Server name"
//	@Param		directory	query		string	false	"Target directory"	default(/)
//	@Param		files		formData	file	true	"Files (multiple allowed)"
//	@Success	204
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/upload [post]
func (a *API) uploadFiles(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	deny, ok := a.denylist(w, r, gs)
	if !ok {
		return
	}
	dir := cleanDir(r.URL.Query().Get("directory"))
	// Streamed straight into the file container, nothing is buffered on disk.
	err := api.Uploads(r, "files", func(name string, body io.Reader) error {
		if !files.ValidName(name) {
			return files.ErrBadName
		}
		file := path.Join(dir, name)
		if err := a.Files.Manager.Write(r.Context(), files.RefOf(gs), file, body, deny); err != nil {
			return err
		}
		a.Audit(r, "file uploaded", "server", gs.Name, "file", file)
		return nil
	})
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// compressFiles godoc
//
//	@Summary		Create a tar.gz archive
//	@Description	Starts a background job (see GET /servers/{server}/jobs); its label is the archive's path.
//	@Tags			Files
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string			true	"Server name"
//	@Param			body	body		FilesRequest	true	"Entries"
//	@Success		202		{object}	files.Job
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/compress [post]
func (a *API) compressFiles(w http.ResponseWriter, r *http.Request) {
	var req FilesRequest
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
	dir := cleanDir(req.Root)
	job, err := a.Files.StartCompress(files.RefOf(gs), dir, req.Files, deny)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "compression started", "server", gs.Name, "folder", dir, "files", req.Files, "archive", job.Label)
	api.JSON(w, http.StatusAccepted, job)
}

// decompressFile godoc
//
//	@Summary		Extract an archive (zip, tar, tar.gz, tar.xz)
//	@Description	Starts a background job (see GET /servers/{server}/jobs); its label is the archive's path.
//	@Tags			Files
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string					true	"Server name"
//	@Param			body	body		DecompressFileRequest	true	"Archive"
//	@Success		202		{object}	files.Job
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/decompress [post]
func (a *API) decompressFile(w http.ResponseWriter, r *http.Request) {
	var req DecompressFileRequest
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
	archive := inRoot(req.Root, req.File)
	job, err := a.Files.StartDecompress(files.RefOf(gs), archive, deny)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "extraction started", "server", gs.Name, "archive", archive)
	api.JSON(w, http.StatusAccepted, job)
}
