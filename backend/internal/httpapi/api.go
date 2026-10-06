// Package httpapi implements the REST and websocket API of the panel.
package httpapi

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/internal/auth"
	"app/internal/clusterinfo"
	"app/internal/console"
	"app/internal/diagnostics"
	"app/internal/egglibrary"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
	"app/internal/schedule"
	"app/internal/selfupgrade"
	"app/internal/serverctl"
	"app/internal/settings"
	"app/internal/users"
)

// API holds the dependencies of all handlers.
type API struct {
	Client client.Client
	// Reader reads directly from the API server, for objects that may have been created
	// a moment ago and are not in the cache yet (e.g. a user right before a server for it).
	Reader client.Reader
	Kube   *kube.Client
	Hub    *console.Hub
	// Files starts the file container on demand, runs file operations (Manager) and the
	// backups, restores and downloads.
	Files   *files.Service
	Opts    gameserver.Options
	Log     *slog.Logger
	Cluster ClusterInfo
	// ClusterService gathers node, hardware and identity details for the admin cluster page.
	ClusterService *clusterinfo.Service
	// Schedules runs server schedules.
	Schedules *schedule.Runner
	// Ops performs power actions and console commands.
	Ops *serverctl.Ops
	// Trigger requests an immediate reconcile of a server.
	Trigger func(namespace, server string)
	// Settings holds the external domain and the enabled storage classes and pools.
	Settings *settings.Store
	// Users creates and changes accounts; Eggs stores eggs (create, import, update, delete).
	Users *users.Store
	Eggs  *eggstore.Store
	// Library lists the eggs of the GitHub repositories in the settings (kept in memory only).
	Library *egglibrary.Library
	// Diagnostics checks why players may not reach a server.
	Diagnostics *diagnostics.Diagnoser
	// SetupToken must accompany the setup request (generated on start while no admin exists).
	SetupToken string
	// DevMode allows websocket connections from the Vite dev server (other port).
	DevMode bool
	// Version is the panel version; UpgradeChecker and Upgrader are nil without self-upgrades.
	Version        string
	UpgradeChecker *selfupgrade.Checker
	Upgrader       *selfupgrade.Upgrader
	// Signer signs session tokens; Limiter throttles failed logins.
	Signer  *auth.Signer
	Limiter *auth.Limiter
	// ClientLimiter throttles failed logins per client across all usernames (password spraying).
	ClientLimiter *auth.Limiter
	// AccountLimiter throttles failed logins per username from all clients (more generous).
	AccountLimiter *auth.Limiter
	// KubeLimiter admits the requests of each user (limitRequests); nil admits every request.
	KubeLimiter *kube.RateLimiter

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

// ErrorResponse is returned for failed requests.
type ErrorResponse struct {
	Error string `json:"error" example:"server not found"`
	// Fields holds validation errors per field or variable.
	Fields map[string]string `json:"fields,omitempty"`
}

// Register adds all routes to the router group (mounted at /api).
func (a *API) Register(r *gin.RouterGroup) {
	a.stats = newStatsCache()
	// Files changed by a background job (backup, restore, download, also scheduled ones) are measured again.
	if a.Files != nil {
		a.Files.Changed = a.stats.disk.Forget
	}

	// Public
	r.POST("/auth/login", a.sameOrigin, a.login)
	r.POST("/auth/logout", a.logout)
	// First start: create the first administrator (only while none exists).
	r.GET("/setup", a.getSetupStatus)
	// Imprint and privacy policy must be readable without an account.
	r.GET("/legal", a.getLegalTexts)
	// Name, logo and favicon: the sign-in page shows them, too.
	r.GET("/branding", a.getBranding)
	r.POST("/setup", a.sameOrigin, a.runSetup)
	// Invite links create an account.
	r.GET("/auth/invite", a.getInvite)
	r.POST("/auth/invite", a.sameOrigin, a.acceptInvite)

	// Every other route needs a session or an API token. The footer polls the request rates: they
	// do not count themselves.
	r.GET("/request-rates", a.authenticate, a.getRequestRates)
	u := r.Group("", a.authenticate, a.limitRequests)
	u.GET("/auth/me", a.getMe)
	u.PUT("/auth/password", a.updatePassword)
	u.POST("/auth/logout-all", a.logoutAll)

	// Everything else waits until a user with a password set by an administrator has chosen their own.
	o := u.Group("", a.requireOwnPassword)
	o.GET("/auth/tokens", a.listTokens)
	o.POST("/auth/tokens", a.createToken)
	o.DELETE("/auth/tokens/:id", a.deleteToken)

	o.GET("/settings", a.getSettings)

	o.GET("/eggs", a.listEggs)
	o.GET("/eggs/:egg", a.getEgg)

	a.registerServers(o)
	a.registerAdmin(o.Group("", a.requireAdmin))
}

// registerServers adds the routes of the servers.
func (a *API) registerServers(u *gin.RouterGroup) {
	// Users only see the servers in their own namespace (see loadServer).
	u.GET("/servers", a.listServers)
	u.GET("/servers/:server", a.getServer)
	u.GET("/servers/:server/stats", a.getServerStats)
	u.GET("/servers/:server/diagnostics", a.getServerDiagnostics)

	// Everything else on a server is blocked for its owner while it is suspended.
	s := u.Group("/servers/:server", a.notSuspended)
	s.PATCH("", a.updateServer)
	s.POST("/power", a.sendPower)
	s.POST("/command", a.sendCommand)
	s.POST("/reinstall", a.reinstallServer)
	s.GET("/ws", a.openConsole)
	s.GET("/schedules", a.listSchedules)
	s.PUT("/schedules", a.updateSchedules)
	s.POST("/schedules/:schedule/run", a.runSchedule)
	s.GET("/jobs", a.listJobs)
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
	adm.PUT("/settings", a.updateSettings)
	adm.GET("/upgrade", a.getUpgradeStatus)
	adm.POST("/upgrade", a.startUpgrade)
	adm.GET("/versions", a.getVersions)
	adm.GET("/settings/storage-classes", a.listStorageClasses)
	adm.GET("/settings/load-balancer-pools", a.listLoadBalancerPools)
	adm.GET("/egg-library", a.listLibraryEggs)
	adm.GET("/egg-library/egg", a.getLibraryEgg)
	adm.POST("/eggs/import", a.importEgg)
	adm.POST("/eggs/import-url", a.importEggURL)
	adm.DELETE("/eggs/:egg", a.deleteEgg)
	adm.POST("/eggs", a.createEgg)
	adm.PUT("/eggs/:egg", a.updateEgg)
	adm.GET("/eggs/:egg/export", a.exportEgg)
	adm.POST("/eggs/:egg/update-from-url", a.updateEggFromURL)
	adm.POST("/servers", a.createServer)
	adm.DELETE("/servers/:server", a.deleteServer)
	adm.POST("/servers/:server/suspend", a.suspendServer)
	adm.POST("/servers/:server/transfer", a.transferServer)
	adm.GET("/users", a.listUsers)
	adm.POST("/users", a.createUser)
	adm.GET("/users/:user", a.getUser)
	adm.PATCH("/users/:user", a.updateUser)
	adm.DELETE("/users/:user", a.deleteUser)
	adm.GET("/invites", a.listInvites)
	adm.POST("/invites", a.createInvite)
	adm.POST("/invites/:invite/renew", a.renewInvite)
	adm.DELETE("/invites/:invite", a.deleteInvite)
}

// audit logs what a user did through the API together with the user ("by") and, for requests
// with an API token, its ID ("token").
func (a *API) audit(c *gin.Context, msg string, args ...any) {
	a.auditAs(principal(c), msg, args...)
}

// auditAs is audit for a caller outside of a request handler (an open console).
func (a *API) auditAs(p *Principal, msg string, args ...any) {
	head := []any{"by", p.User.Name}
	if p.TokenID != "" {
		head = append(head, "token", p.TokenID)
	}
	a.Log.Info(msg, append(head, args...)...)
}
