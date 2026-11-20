package httpapi

import (
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/base/api"
	"app/internal/files"
	"app/internal/serverctl"
)

// Backup is an archive in the backup folder of a server.
type Backup struct {
	Name      string    `json:"name"      example:"backup-2026-09-29_040000.tar.gz"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
	// Path is the file manager path (download with /files/download?file=…).
	Path string `json:"path" example:"/.backups/backup-2026-09-29_040000.tar.gz"`
}

// BackupList lists the backups of a server.
type BackupList struct {
	Items []Backup `json:"items"`
}

// CreateBackupRequest starts a backup.
type CreateBackupRequest struct {
	// Label is appended to the archive name.
	Label string `json:"label,omitempty" example:"before-update"`
}

// listBackups godoc
//
//	@Summary		Backups of a server
//	@Description	Archives in the folder .backups of the server volume, newest first.
//	@Tags			Backups
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Success		200		{object}	BackupList
//	@Security		BearerAuth
//	@Router			/servers/{server}/backups [get]
func (a *API) listBackups(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	entries, err := a.Files.Manager.ListBackups(r.Context(), files.RefOf(gs))
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	out := BackupList{Items: []Backup{}}
	for _, e := range entries {
		out.Items = append(
			out.Items,
			Backup{Name: e.Name, Size: e.Size, CreatedAt: e.ModifiedAt, Path: files.BackupPath(e.Name)},
		)
	}
	api.JSON(w, http.StatusOK, out)
}

// createBackup godoc
//
//	@Summary		Create a backup
//	@Description	Packs all server files except the backup folder into a tar.gz in the background. Works while the
//	@Description	server runs.
//	@Tags			Backups
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string				true	"Server name"
//	@Param			body	body		CreateBackupRequest	false	"Label"
//	@Success		202		{object}	files.Job
//	@Failure		409		{object}	api.ErrorResponse	"a backup or restore is running"
//	@Security		BearerAuth
//	@Router			/servers/{server}/backups [post]
func (a *API) createBackup(w http.ResponseWriter, r *http.Request) {
	var req CreateBackupRequest
	if !a.DecodeOptional(w, r, &req) {
		return
	}
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	job, err := a.Files.StartBackup(files.RefOf(gs), req.Label, time.Now().In(a.Schedules.Location))
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Hub.Daemon(gs.Name, "Creating backup "+job.Label+"...")
	a.Audit(r, "backup started", "server", gs.Name, "backup", job.Label)
	api.JSON(w, http.StatusAccepted, job)
}

// restoreBackup godoc
//
//	@Summary		Restore a backup
//	@Description	Deletes all server files except the backup folder and extracts the backup. The server must be
//	@Description	stopped; it cannot be started until the restore has finished.
//	@Tags			Backups
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Param			backup	path		string	true	"Backup file name"
//	@Success		202		{object}	files.Job
//	@Failure		409		{object}	api.ErrorResponse	"server running or job in progress"
//	@Security		BearerAuth
//	@Router			/servers/{server}/backups/{backup}/restore [post]
func (a *API) restoreBackup(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	// Uncached: a server started a moment ago must count as running.
	cur := &v1alpha1.GameServer{}
	if err := a.Reader.Get(r.Context(), client.ObjectKeyFromObject(gs), cur); err != nil {
		a.Fail(w, r, err)
		return
	}
	if err := serverctl.Stopped(cur); err != nil {
		a.Fail(w, r, err)
		return
	}
	name := r.PathValue("backup")
	job, err := a.Files.StartRestore(files.RefOf(gs), name)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Hub.Daemon(gs.Name, "Restoring backup "+name+"...")
	a.Audit(r, "backup restore started", "server", gs.Name, "backup", name)
	api.JSON(w, http.StatusAccepted, job)
}

// deleteBackup godoc
//
//	@Summary	Delete a backup
//	@Tags		Backups
//	@Param		server	path	string	true	"Server name"
//	@Param		backup	path	string	true	"Backup file name"
//	@Success	204
//	@Security	BearerAuth
//	@Router		/servers/{server}/backups/{backup} [delete]
func (a *API) deleteBackup(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	name := r.PathValue("backup")
	if err := a.Files.DeleteBackup(r.Context(), files.RefOf(gs), name); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "backup deleted", "server", gs.Name, "backup", name)
	w.WriteHeader(http.StatusNoContent)
}
