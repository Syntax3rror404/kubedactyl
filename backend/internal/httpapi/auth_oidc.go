package httpapi

import (
	"cmp"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/sso"
	"app/internal/users"
)

// oidcCookie keeps a sign-in through the identity provider between the redirect and the callback (signed). It is
// SameSite=Lax: the callback is a navigation from the identity provider's site, which does not carry Strict
// cookies. Over HTTPS it is a __Host- cookie (path /, no domain), which a site on a sibling subdomain cannot
// set; otherwise it is sent to oidcPath only.
const (
	oidcCookie       = "kd_oidc"
	oidcSecureCookie = "__Host-kd_oidc"
	oidcPath         = "/api/auth/oidc"
)

// Why a sign-in through the identity provider failed, for the sign-in page (/login?sso=<reason>); the panel log
// has the details.
const (
	ssoNoAccess = "access"
	ssoAccount  = "account"
	ssoFailed   = "failed"
)

var errOIDCOff = errors.New("single sign-on is off")

// OIDCSignIn tells the sign-in page whether to offer single sign-on.
type OIDCSignIn struct {
	Enabled bool `json:"enabled"`
	// Name is shown on the button: "Sign in with <name>".
	Name string `json:"name" example:"Keycloak"`
}

// getOIDCSignIn godoc
//
//	@Summary		Single sign-on of the sign-in page
//	@Description	Public: whether users can sign in through an OpenID Connect identity provider.
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	OIDCSignIn
//	@Router			/auth/oidc [get]
func (a *API) getOIDCSignIn(c *gin.Context) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, OIDCSignIn{Enabled: spec.OIDC.Enabled, Name: cmp.Or(spec.OIDC.Name, "SSO")})
}

// startOIDC godoc
//
//	@Summary		Sign in through the identity provider
//	@Description	Public, opened by the browser: redirects to the sign-in page of the identity provider, which
//	@Description	returns to /auth/oidc/callback. Failures redirect to /login?sso=failed.
//	@Tags			Auth
//	@Param			next	query	string	false	"Page of the panel to open after the sign-in"
//	@Success		302
//	@Router			/auth/oidc/start [get]
func (a *API) startOIDC(c *gin.Context) {
	cfg, err := a.oidcConfig(c)
	var authURL string
	var flow sso.Flow
	if err == nil {
		authURL, flow, err = a.oidc.Start(c.Request.Context(), cfg, localPath(c.Query("next")))
	}
	if err == nil {
		err = a.setOIDCFlow(c, flow)
	}
	if err != nil {
		a.oidcFailed(c, err)
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

// finishOIDC godoc
//
//	@Summary		Callback of the identity provider
//	@Description	Public, opened by the browser: verifies the sign-in, starts a session (cookie) and redirects to
//	@Description	the page the sign-in started from. The account is the one linked to the user of the identity
//	@Description	provider, else the one with the same username, else a new one. Failures redirect to
//	@Description	/login?sso=<reason> (access: in no group with access or disabled, account: the account cannot be
//	@Description	used, failed).
//	@Tags			Auth
//	@Param			code	query	string	false	"Authorization code"
//	@Param			state	query	string	false	"State of the sign-in"
//	@Success		302
//	@Router			/auth/oidc/callback [get]
func (a *API) finishOIDC(c *gin.Context) {
	flow := a.takeOIDCFlow(c)
	user, err := a.oidcUser(c, flow)
	if err == nil {
		_, err = a.startSession(c, user)
	}
	if err != nil {
		a.oidcFailed(c, err)
		return
	}
	a.Log.Info("signed in", "user", user.Name, "ip", c.ClientIP(), "via", "oidc")
	c.Redirect(http.StatusFound, flow.Next)
}

// oidcUser verifies the callback and returns the account of the user the identity provider signed in. Errors
// name that user when it is known (for the log).
func (a *API) oidcUser(c *gin.Context, flow sso.Flow) (*v1alpha1.User, error) {
	if e := c.Query("error"); e != "" {
		return nil, fmt.Errorf("identity provider: %s: %s", e, c.Query("error_description"))
	}
	cfg, err := a.oidcConfig(c)
	if err != nil {
		return nil, err
	}
	acc, err := a.oidc.Finish(c.Request.Context(), cfg, flow, c.Query("state"), c.Query("code"))
	var user *v1alpha1.User
	if err == nil {
		user, err = a.Users.SignInOIDC(c, acc, cfg.OIDCSettings)
	}
	if err != nil && acc.Identity.Subject != "" {
		err = fmt.Errorf("user %q (subject %s): %w", acc.Username, acc.Identity.Subject, err)
	}
	return user, err
}

// oidcConfig is the client configuration of the identity provider.
func (a *API) oidcConfig(c *gin.Context) (sso.Config, error) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		return sso.Config{}, err
	}
	if !spec.OIDC.Enabled {
		return sso.Config{}, errOIDCOff
	}
	secret, err := a.Settings.OIDCClientSecret(c)
	return sso.Config{OIDCSettings: spec.OIDC, ClientSecret: secret}, err
}

// oidcFailed logs why a sign-in through the identity provider failed and sends the browser back to the sign-in
// page with the reason.
func (a *API) oidcFailed(c *gin.Context, err error) {
	reason := ssoFailed
	switch {
	case errors.Is(err, sso.ErrNoAccess), errors.Is(err, users.ErrDisabled):
		reason = ssoNoAccess
	case errors.Is(err, users.ErrOtherIdentity), errors.Is(err, users.ErrNotLinked),
		errors.Is(err, users.ErrInvalidUsername), errors.Is(err, users.ErrLastAdmin):
		reason = ssoAccount
	}
	a.Log.Warn("single sign-on failed", "ip", c.ClientIP(), "err", err)
	c.Redirect(http.StatusFound, "/login?sso="+reason)
}

// flowCookie is the cookie of a sign-in through the identity provider, without value.
func flowCookie(c *gin.Context) *http.Cookie {
	if auth.HTTPS(c.Request) {
		return &http.Cookie{
			Name: oidcSecureCookie, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		}
	}
	return &http.Cookie{Name: oidcCookie, Path: oidcPath, HttpOnly: true, SameSite: http.SameSiteLaxMode}
}

func (a *API) setOIDCFlow(c *gin.Context, flow sso.Flow) error {
	cookie := flowCookie(c)
	cookie.Expires = time.Now().Add(sso.FlowLifetime)
	value, err := a.Signer.SignValue(oidcCookie, flow, cookie.Expires)
	cookie.Value = value
	http.SetCookie(c.Writer, cookie)
	return err
}

// takeOIDCFlow returns the sign-in this browser started (empty when none) and removes its cookie.
func (a *API) takeOIDCFlow(c *gin.Context) sso.Flow {
	cookie, flow := flowCookie(c), sso.Flow{Next: "/"}
	if value, err := c.Cookie(cookie.Name); err == nil {
		_ = a.Signer.VerifyValue(oidcCookie, value, &flow, time.Now())
	}
	cookie.MaxAge = -1
	http.SetCookie(c.Writer, cookie)
	return flow
}

// localPath returns next when it is a page of the panel ("/" else, also for API paths), so the redirect after a
// sign-in never leaves the web interface.
func localPath(next string) string {
	u, err := url.Parse(next)
	if err != nil || u.Scheme != "" || u.Host != "" || !strings.HasPrefix(next, "/") ||
		strings.HasPrefix(next, "//") || strings.Contains(next, `\`) || strings.HasPrefix(u.Path, "/api/") {
		return "/"
	}
	return next
}
