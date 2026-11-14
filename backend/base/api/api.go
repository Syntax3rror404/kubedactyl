// Package api is the HTTP API every app shares: sign-in (password, single sign-on, setup, invites), the own
// account, users and invites, the panel settings and upgrades. The app embeds API in its own and adds its
// routes to the groups Register returns; App adds its parts to the answers.
package api

import (
	"context"
	"log/slog"
	"sync"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/auth"
	"app/base/kube"
	"app/base/selfupgrade"
	"app/base/settings"
	"app/base/users"
)

// App is what the app adds to the answers of the base routes; the app's API implements it.
type App interface {
	// UserViews describes the app's part of the accounts, in the order of list.
	UserViews(ctx context.Context, list []*v1alpha1.User) []app.UserView
	// InviteView describes the app's part of an invite.
	InviteView(i *v1alpha1.Invite) app.InviteView
	// RequestRates are the app's request rates for the caller.
	RequestRates(p *Principal) app.RequestRates
	// StatusOf is the HTTP status of an error of the app's domain packages (0: none of them).
	StatusOf(err error) int
}

// API holds the dependencies of the base handlers.
type API struct {
	Client client.Client
	// Reader reads directly from the API server, for objects that may have been created a moment ago.
	Reader client.Reader
	// Namespace is the panel namespace (users, invites, settings, session key).
	Namespace string
	Log       *slog.Logger
	// App adds the app's parts to the answers.
	App App
	// Settings reads and saves the panel settings; Users creates and changes accounts.
	Settings *settings.Store
	Users    *users.Store
	// SetupToken must accompany the setup request (generated on start while no admin exists).
	SetupToken string
	// DevMode allows requests from the Vite dev server (other port).
	DevMode bool
	// Version is the panel version; UpgradeChecker and Upgrader are nil without self-upgrades.
	Version        string
	UpgradeChecker *selfupgrade.Checker
	Upgrader       *selfupgrade.Upgrader
	// Signer signs session tokens; Limiter throttles failed logins per client and username.
	Signer  *auth.Signer
	Limiter *auth.Limiter
	// ClientLimiter throttles failed logins per client across all usernames (password spraying).
	ClientLimiter *auth.Limiter
	// AccountLimiter throttles failed logins per username from all clients (more generous).
	AccountLimiter *auth.Limiter
	// KubeLimiter admits the requests of each user (limitRequests); nil admits every request.
	KubeLimiter *kube.RateLimiter

	// oidc signs users in through the identity provider of the settings.
	oidc auth.OIDC
	// setupMu serializes setup requests, so only one first administrator can be created.
	setupMu sync.Mutex
}

// ErrorResponse is returned for failed requests.
type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
	// Fields holds validation errors per field or variable.
	Fields map[string]string `json:"fields,omitempty"`
}

// Groups are the route groups the app adds its routes to.
type Groups struct {
	// Public needs no sign-in.
	Public *gin.RouterGroup
	// Signed needs a session or an API token of a user who has chosen their own password.
	Signed *gin.RouterGroup
	// Admin is Signed for administrators.
	Admin *gin.RouterGroup
}

// Register adds the base routes to the router group (mounted at /api) and returns the groups for the app's
// routes. The routes of users and invites the app adds with RegisterAccounts.
func (a *API) Register(r *gin.RouterGroup) Groups {
	// Public
	r.POST("/auth/login", a.sameOrigin, a.login)
	r.POST("/auth/logout", a.notCrossSite, a.logout)
	// First start: create the first administrator (only while none exists).
	r.GET("/setup", a.getSetupStatus)
	r.POST("/setup", a.sameOrigin, a.runSetup)
	// Imprint and privacy policy must be readable without an account.
	r.GET("/legal", a.getLegalTexts)
	// Name, logo and favicon: the sign-in page shows them, too.
	r.GET("/branding", a.getBranding)
	// Invite links create an account.
	r.GET("/auth/invite", a.getInvite)
	r.POST("/auth/invite", a.sameOrigin, a.acceptInvite)
	// Single sign-on through an OpenID Connect identity provider (start and callback are browser redirects).
	r.GET("/auth/oidc", a.getOIDCSignIn)
	r.GET("/auth/oidc/start", a.startOIDC)
	r.GET("/auth/oidc/callback", a.finishOIDC)

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

	adm := o.Group("", a.RequireAdmin)
	adm.PUT("/settings", a.updateSettings)
	adm.GET("/upgrade", a.getUpgradeStatus)
	adm.POST("/upgrade", a.startUpgrade)
	adm.GET("/versions", a.getVersions)
	adm.POST("/users", a.createUser)
	adm.GET("/users/:user", a.getUser)
	adm.PATCH("/users/:user", a.updateUser)
	return Groups{Public: r, Signed: o, Admin: adm}
}

// RegisterAccounts adds the routes that list and delete users and manage invites to m: the Admin group, or a
// group that also lets users manage the accounts they own (users.App.Owner).
func (a *API) RegisterAccounts(m *gin.RouterGroup) {
	m.GET("/users", a.listUsers)
	m.DELETE("/users/:user", a.deleteUser)
	m.GET("/invites", a.listInvites)
	m.POST("/invites", a.createInvite)
	m.POST("/invites/:invite/renew", a.renewInvite)
	m.DELETE("/invites/:invite", a.deleteInvite)
}

// Audit logs what a user did through the API together with the user ("by") and, for requests
// with an API token, its ID ("token").
func (a *API) Audit(c *gin.Context, msg string, args ...any) {
	a.AuditAs(PrincipalOf(c), msg, args...)
}

// AuditAs is Audit for a caller outside of a request handler (an open websocket).
func (a *API) AuditAs(p *Principal, msg string, args ...any) {
	head := []any{"by", p.User.Name}
	if p.TokenID != "" {
		head = append(head, "token", p.TokenID)
	}
	a.Log.Info(msg, append(head, args...)...)
}
