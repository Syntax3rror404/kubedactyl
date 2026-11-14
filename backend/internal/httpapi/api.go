// Package httpapi implements the REST and websocket API of the panel.
package httpapi

import (
	"github.com/gin-gonic/gin"

	"app/base/api"
	"app/base/kube"
	"app/internal/clusterinfo"
	"app/internal/console"
	"app/internal/diagnostics"
	"app/internal/egglibrary"
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
	Library *egglibrary.Library
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

// Register adds all routes to the router group (mounted at /api): the base routes, then the servers, eggs and
// the cluster.
func (a *API) Register(r *gin.RouterGroup) {
	a.stats = newStatsCache()
	// Files changed by a background job (backup, restore, download, also scheduled ones) are measured again.
	if a.Files != nil {
		a.Files.Changed = a.stats.disk.Forget
	}
	a.API.App = a
	g := a.API.Register(r)
	g.Signed.GET("/eggs", a.listEggs)
	g.Signed.GET("/eggs/:egg", a.getEgg)
	a.registerServers(g.Signed)
	// Administrators manage all accounts and invites, users their subusers.
	a.RegisterAccounts(g.Signed.Group("", a.requireNoSubuser))
	a.registerAdmin(g.Admin)
}

// registerServers adds the routes of the servers.
func (a *API) registerServers(u *gin.RouterGroup) {
	// Users only see the servers in their own namespace, subusers the ones shared with them (see loadServer).
	u.GET("/servers", a.listServers)
	u.GET("/servers/:server", a.getServer)
	u.GET("/servers/:server/stats", a.getServerStats)
	u.GET("/servers/:server/diagnostics", a.requireAccess(accessFull), a.getServerDiagnostics)

	// Everything else on a server is blocked while it is being removed, and for its owner while it is suspended.
	l := u.Group("/servers/:server", a.notLocked)
	l.GET("/ws", a.openConsole)
	l.POST("/power", a.requireAccess(accessControl), a.sendPower)
	l.POST("/command", a.requireAccess(accessControl), a.sendCommand)

	// The rest is for the owner (and administrators), never for subusers.
	s := l.Group("", a.requireAccess(accessFull))
	s.PATCH("", a.updateServer)
	s.POST("/reinstall", a.reinstallServer)
	s.GET("/schedules", a.listSchedules)
	s.PUT("/schedules", a.updateSchedules)
	s.POST("/schedules/:schedule/run", a.runSchedule)
	s.GET("/jobs", a.listJobs)
	s.POST("/jobs/:job/cancel", a.cancelJob)
	s.POST("/backups", a.createBackup)
	s.POST("/backups/:backup/restore", a.restoreBackup)
	b := s.Group("/backups", a.requireFilesPod, a.remeasureDisk)
	b.GET("", a.listBackups)
	b.DELETE("/:backup", a.deleteBackup)

	s.GET("/files/session", a.getFilesSession)
	s.POST("/files/session", a.openFilesSession)
	f := s.Group("/files", a.requireFilesPod, a.remeasureDisk)
	f.GET("/list", a.listFiles)
	f.GET("/contents", a.readFile)
	f.GET("/download", a.downloadFile)
	f.POST("/write", a.writeFile)
	f.POST("/create-folder", a.createFolder)
	f.POST("/delete", a.deleteFiles)
	f.PUT("/rename", a.renameFile)
	f.POST("/upload", a.uploadFiles)
	f.POST("/compress", a.compressFiles)
	f.POST("/decompress", a.decompressFile)
	f.POST("/pull", a.pullFile)
}

// registerAdmin adds the routes only administrators may use.
func (a *API) registerAdmin(adm *gin.RouterGroup) {
	adm.GET("/cluster", a.getClusterInfo)
	adm.GET("/cluster/nodes", a.listClusterNodes)
	adm.POST("/cluster/nodes/probe", a.probeClusterNodes)
	adm.GET("/cluster/identity", a.getClusterIdentity)
	adm.GET("/cluster/health", a.getClusterHealth)
	adm.GET("/settings/storage-classes", a.listStorageClasses)
	adm.GET("/settings/load-balancer-pools", a.listLoadBalancerPools)
	adm.GET("/egg-library", a.listLibraryEggs)
	adm.GET("/egg-library/egg", a.getLibraryEgg)
	adm.GET("/egg-library/repository", a.getLibraryRepository)
	adm.POST("/eggs/import", a.importEgg)
	adm.POST("/eggs/import-url", a.importEggURL)
	adm.DELETE("/eggs/:egg", a.deleteEgg)
	adm.POST("/eggs", a.createEgg)
	adm.PUT("/eggs/:egg", a.updateEgg)
	adm.GET("/eggs/:egg/export", a.exportEgg)
	adm.POST("/eggs/:egg/update-from-url", a.updateEggFromURL)
	adm.POST("/servers", a.createServer)
	adm.DELETE("/servers/:server", a.deleteServer)
	adm.POST("/servers/:server/suspend", a.notLocked, a.suspendServer)
	adm.POST("/servers/:server/transfer", a.notLocked, a.transferServer)
	adm.POST("/servers/:server/migrate", a.notLocked, a.migrateServer)
	adm.POST("/servers/:server/migrate/cancel", a.cancelMigration)
}
