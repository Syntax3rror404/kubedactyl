package httpapi

import (
	"io"
	"net/http"
	"path"
	"strings"

	"app/internal/files"

	"github.com/gin-gonic/gin"
)

// CreateFolderRequest creates a directory.
type CreateFolderRequest struct {
	Root string `json:"root,omitempty" example:"/"`
	Name string `json:"name"           example:"plugins" binding:"required"`
}

// FilesRequest names entries inside a directory (delete, compress).
type FilesRequest struct {
	Root  string   `json:"root,omitempty" example:"/"`
	Files []string `json:"files"          example:"world,logs" binding:"required"`
}

// RenameFileRequest moves a file or directory.
type RenameFileRequest struct {
	Root string `json:"root,omitempty" example:"/"`
	From string `json:"from"           example:"old.txt" binding:"required"`
	To   string `json:"to"             example:"new.txt" binding:"required"`
}

// DecompressFileRequest extracts an archive.
type DecompressFileRequest struct {
	Root string `json:"root,omitempty" example:"/"`
	File string `json:"file"           example:"world.zip" binding:"required"`
}

// CompressResponse names the created archive.
type CompressResponse struct {
	Name string `json:"name" example:"archive-2026-09-28T120000.tar.gz"`
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
func (a *API) writeFile(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	err := a.Files.Manager.Write(c, files.RefOf(gs), c.Query("file"), c.Request.Body, a.denylist(c, gs))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "file written", "server", gs.Name, "file", c.Query("file"))
	c.Status(http.StatusNoContent)
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
func (a *API) createFolder(c *gin.Context) {
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	folder := inRoot(req.Root, req.Name)
	if err := a.Files.Manager.CreateFolder(c, files.RefOf(gs), folder, a.denylist(c, gs)); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "folder created", "server", gs.Name, "folder", folder)
	c.Status(http.StatusNoContent)
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
func (a *API) deleteFiles(c *gin.Context) {
	var req FilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	var paths []string
	for _, f := range req.Files {
		if !files.ValidName(f) {
			a.fail(c, files.ErrBadName)
			return
		}
		paths = append(paths, inRoot(req.Root, f))
	}
	if err := a.Files.Manager.Delete(c, files.RefOf(gs), paths, a.denylist(c, gs)); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "files deleted", "server", gs.Name, "files", paths)
	c.Status(http.StatusNoContent)
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
func (a *API) renameFile(c *gin.Context) {
	var req RenameFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	from := inRoot(req.Root, req.From)
	to := inRoot(req.Root, req.To)
	if err := a.Files.Manager.Rename(c, files.RefOf(gs), from, to, a.denylist(c, gs)); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "file renamed", "server", gs.Name, "from", from, "to", to)
	c.Status(http.StatusNoContent)
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
func (a *API) uploadFiles(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	deny := a.denylist(c, gs)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		a.fail(c, badRequest(err))
		return
	}
	dir := cleanDir(c.Query("directory"))
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			a.fail(c, badRequest(err))
			return
		}
		name := path.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
		if part.FormName() != "files" || !files.ValidName(name) {
			part.Close()
			continue
		}
		// Streamed straight into the helper pod, nothing is buffered on disk.
		err = a.Files.Manager.Write(c, files.RefOf(gs), path.Join(dir, name), part, deny)
		part.Close()
		if err != nil {
			a.fail(c, err)
			return
		}
		a.audit(c, "file uploaded", "server", gs.Name, "file", path.Join(dir, name))
	}
	c.Status(http.StatusNoContent)
}

// compressFiles godoc
//
//	@Summary	Create a tar.gz archive
//	@Tags		Files
//	@Accept		json
//	@Produce	json
//	@Param		server	path		string			true	"Server name"
//	@Param		body	body		FilesRequest	true	"Entries"
//	@Success	200		{object}	CompressResponse
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/compress [post]
func (a *API) compressFiles(c *gin.Context) {
	var req FilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	name, err := a.Files.Manager.Compress(c, files.RefOf(gs), cleanDir(req.Root), req.Files, a.denylist(c, gs))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "files compressed", "server", gs.Name, "folder", cleanDir(req.Root), "files", req.Files, "archive", name)
	c.JSON(http.StatusOK, CompressResponse{Name: name})
}

// decompressFile godoc
//
//	@Summary	Extract an archive (zip, tar, tar.gz, tar.xz)
//	@Tags		Files
//	@Accept		json
//	@Param		server	path	string					true	"Server name"
//	@Param		body	body	DecompressFileRequest	true	"Archive"
//	@Success	204
//	@Security	BearerAuth
//	@Router		/servers/{server}/files/decompress [post]
func (a *API) decompressFile(c *gin.Context) {
	var req DecompressFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	archive := inRoot(req.Root, req.File)
	if err := a.Files.Manager.Decompress(c, files.RefOf(gs), archive, a.denylist(c, gs)); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "archive extracted", "server", gs.Name, "archive", archive)
	c.Status(http.StatusNoContent)
}
