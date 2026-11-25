// Package httpapi implements the REST and websocket API of the panel.
package httpapi

import (
	"app/base/api"
	"app/base/kube"
	"app/internal/clusterinfo"
	"app/internal/console"
	"app/internal/diagnostics"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/schedule"
	"app/internal/serverctl"
)

// API holds the dependencies of all handlers: the base API (sign-in, accounts, settings, upgrades) and the
// services of the servers, eggs and the cluster.
type API struct {
	*api.API
	Kube *kube.Client
	Hub  *console.Hub
	// Files starts the file container on demand, runs file operations (Manager) and the
	// backups, restores and downloads.
	Files   *files.Service
	Opts    gameserver.Options
	Cluster ClusterInfo
	// ClusterService gathers node, hardware and identity details for the admin cluster page.
	ClusterService *clusterinfo.Service
	// Schedules runs server schedules.
	Schedules *schedule.Runner
	// Ops performs the actions on servers (power, commands, reinstall, transfer, migration).
	Ops *serverctl.Ops
	// Trigger requests an immediate reconcile of a server.
	Trigger func(namespace, server string)
	// Eggs stores eggs (create, import, update, delete).
	Eggs *eggstore.Store
	// Library lists the eggs of the git repositories in the settings (kept in memory only).
	Library *eggstore.Library
	// Diagnostics checks why players may not reach a server.
	Diagnostics *diagnostics.Diagnoser

	stats *statsCache
}

// ClusterInfo describes the cluster the panel is connected to.
type ClusterInfo struct {
	Context   string `json:"context"   example:"my-cluster"`
	Server    string `json:"server"    example:"https://k8s.example.com:6443"`
	Version   string `json:"version"   example:"v1.36.4"`
	Namespace string `json:"namespace" example:"kubedactyl"`
	// StorageClass and LoadBalancerPool are the defaults of the panel settings.
	StorageClass     string `json:"storageClass"     example:"longhorn"`
	LoadBalancerPool string `json:"loadBalancerPool" example:"general-pool"`
}

// Register adds all routes to m (mounted at /api): the base routes, then the servers, eggs and the cluster.
func (a *API) Register(m *api.Mux) {
	a.stats = newStatsCache()
	// Files changed by a background job (backup, restore, download, also scheduled ones) are measured again.
	if a.Files != nil {
		a.Files.Changed = a.stats.disk.Forget
	}
	a.API.App = a
	g := a.API.Register(m)
	g.Signed.Handle("GET /eggs", a.listEggs)
	g.Signed.Handle("GET /eggs/{egg}", a.getEgg)
	a.registerServers(g.Signed)
	// Administrators manage all accounts and invites, users their subusers.
	a.RegisterAccounts(g.Signed.With(a.Guard(requireNoSubuser)))
	a.registerAdmin(g.Admin)
}

// registerServers adds the routes of the servers. Each group adds a check to the one before:
//  1. Read: users see the servers in their own namespace, subusers the ones shared with them (loadServer).
//  2. Not locked: blocked while the server is being removed, and for its owner while it is suspended.
//  3. Control: power and commands, also for subusers with the control share.
//  4. Full access: the owner and administrators, never subusers.
//  5. Files: also needs the file container.
func (a *API) registerServers(u *api.Mux) {
	// 1. Read
	u.Handle("GET /servers", a.listServers)
	u.Handle("GET /servers/{server}", a.getServer)
	u.Handle("GET /servers/{server}/stats", a.getServerStats)
	u.With(a.requireAccess(accessFull)).Handle("GET /servers/{server}/diagnostics", a.getServerDiagnostics)

	// 2. Not locked
	l := u.With(a.notLocked)
	l.Handle("GET /servers/{server}/ws", a.openConsole)
	// 3. Control
	c := l.With(a.requireAccess(accessControl))
	c.Handle("POST /servers/{server}/power", a.sendPower)
	c.Handle("POST /servers/{server}/command", a.sendCommand)

	// 4. Full access
	s := l.With(a.requireAccess(accessFull))
	s.Handle("PATCH /servers/{server}", a.updateServer)
	s.Handle("POST /servers/{server}/reinstall", a.reinstallServer)
	s.Handle("GET /servers/{server}/schedules", a.listSchedules)
	s.Handle("PUT /servers/{server}/schedules", a.updateSchedules)
	s.Handle("POST /servers/{server}/schedules/{schedule}/run", a.runSchedule)
	s.Handle("GET /servers/{server}/jobs", a.listJobs)
	s.Handle("POST /servers/{server}/jobs/{job}/cancel", a.cancelJob)
	s.Handle("POST /servers/{server}/backups", a.createBackup)
	s.Handle("POST /servers/{server}/backups/{backup}/restore", a.restoreBackup)
	s.Handle("GET /servers/{server}/files/session", a.getFilesSession)
	s.Handle("POST /servers/{server}/files/session", a.openFilesSession)

	// 5. Files
	f := s.With(a.requireFilesPod, a.remeasureDisk)
	f.Handle("GET /servers/{server}/backups", a.listBackups)
	f.Handle("DELETE /servers/{server}/backups/{backup}", a.deleteBackup)
	f.Handle("GET /servers/{server}/files/list", a.listFiles)
	f.Handle("GET /servers/{server}/files/contents", a.readFile)
	f.Handle("GET /servers/{server}/files/download", a.downloadFile)
	f.Handle("POST /servers/{server}/files/write", a.writeFile)
	f.Handle("POST /servers/{server}/files/create-folder", a.createFolder)
	f.Handle("POST /servers/{server}/files/delete", a.deleteFiles)
	f.Handle("PUT /servers/{server}/files/rename", a.renameFile)
	f.Handle("POST /servers/{server}/files/upload", a.uploadFiles)
	f.Handle("POST /servers/{server}/files/compress", a.compressFiles)
	f.Handle("POST /servers/{server}/files/decompress", a.decompressFile)
	f.Handle("POST /servers/{server}/files/pull", a.pullFile)
}

// registerAdmin adds the routes only administrators may use.
func (a *API) registerAdmin(adm *api.Mux) {
	adm.Handle("GET /cluster", a.getClusterInfo)
	adm.Handle("GET /cluster/nodes", a.listClusterNodes)
	adm.Handle("POST /cluster/nodes/probe", a.probeClusterNodes)
	adm.Handle("GET /cluster/identity", a.getClusterIdentity)
	adm.Handle("GET /cluster/health", a.getClusterHealth)
	adm.Handle("GET /settings/storage-classes", a.listStorageClasses)
	adm.Handle("GET /settings/load-balancer-pools", a.listLoadBalancerPools)
	adm.Handle("GET /egg-library", a.listLibraryEggs)
	adm.Handle("GET /egg-library/egg", a.getLibraryEgg)
	adm.Handle("GET /egg-library/repository", a.getLibraryRepository)
	adm.Handle("POST /eggs/import", a.importEgg)
	adm.Handle("POST /eggs/import-url", a.importEggURL)
	adm.Handle("DELETE /eggs/{egg}", a.deleteEgg)
	adm.Handle("POST /eggs", a.createEgg)
	adm.Handle("PUT /eggs/{egg}", a.updateEgg)
	adm.Handle("GET /eggs/{egg}/export", a.exportEgg)
	adm.Handle("POST /eggs/{egg}/update-from-url", a.updateEggFromURL)
	adm.Handle("POST /servers", a.createServer)
	adm.Handle("DELETE /servers/{server}", a.deleteServer)
	locked := adm.With(a.notLocked)
	locked.Handle("POST /servers/{server}/suspend", a.suspendServer)
	locked.Handle("POST /servers/{server}/transfer", a.transferServer)
	locked.Handle("POST /servers/{server}/migrate", a.migrateServer)
	adm.Handle("POST /servers/{server}/migrate/cancel", a.cancelMigration)
}
