package api

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/app"
	"app/base/kube"
)

// RequestRates are the requests per second of the signed-in user and, for administrators, the
// Kubernetes API calls of the whole panel, each with its limit (settings "Kube API limit"), and the rates of
// the app.
type RequestRates struct {
	User  kube.Rate  `json:"user"`
	Panel *kube.Rate `json:"panel,omitempty" extensions:"x-nullable"`
	app.RequestRates
}

// getRequestRates godoc
//
//	@Summary		Request rates
//	@Description	Averages over the last 5 seconds; refused requests and waiting calls count, too. This request
//	@Description	does not count. Panel and the rates of the app are set for administrators only.
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	RequestRates
//	@Security		BearerAuth
//	@Router			/request-rates [get]
func (a *API) getRequestRates(c *gin.Context) {
	var out RequestRates
	p := PrincipalOf(c)
	if a.KubeLimiter != nil {
		out.User = a.KubeLimiter.UserRate(p.User.Name)
		if p.Admin() {
			panel := a.KubeLimiter.Rate()
			out.Panel = &panel
		}
	}
	if a.App != nil {
		out.RequestRates = a.App.RequestRates(p)
	}
	c.JSON(http.StatusOK, out)
}

// SettingsView is the panel settings; the client secret of the identity provider is never sent. Users get only
// the fields the web UI shows them, the others are empty.
type SettingsView struct {
	v1alpha1.PanelSettingsSpec
	// OIDCClientSecretSet tells administrators whether a client secret is stored (always false for users).
	OIDCClientSecretSet bool `json:"oidcClientSecretSet"`
}

// UpdateSettingsRequest is the panel settings with the client secret of the identity provider.
type UpdateSettingsRequest struct {
	v1alpha1.PanelSettingsSpec
	// OIDCClientSecret replaces the stored client secret (empty removes it); omitted keeps it.
	OIDCClientSecret *string `json:"oidcClientSecret,omitempty" extensions:"x-nullable"`
}

// sendSettings answers with the settings and, for administrators (only they edit them), whether a client
// secret is stored; users get settings.Store.ForUsers.
func (a *API) sendSettings(c *gin.Context, spec v1alpha1.PanelSettingsSpec) {
	if !PrincipalOf(c).Admin() {
		c.JSON(http.StatusOK, SettingsView{PanelSettingsSpec: a.Settings.ForUsers(spec)})
		return
	}
	secret, err := a.Settings.OIDCClientSecret(c)
	if err != nil {
		a.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, SettingsView{PanelSettingsSpec: spec, OIDCClientSecretSet: secret != ""})
}

// getSettings godoc
//
//	@Summary		Panel settings
//	@Description	All settings for administrators (single sign-on without its client secret). Other users get only
//	@Description	the API docs switch, their limit for API tokens and the fields of the app the web UI shows them.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	SettingsView
//	@Security		BearerAuth
//	@Router			/settings [get]
func (a *API) getSettings(c *gin.Context) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		a.Fail(c, err)
		return
	}
	a.sendSettings(c, spec)
}

// updateSettings godoc
//
//	@Summary		Update the panel settings
//	@Description	Every wrong field is reported at once (422 with the fields). While single sign-on is on, the
//	@Description	identity provider must answer.
//	@Tags			Settings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UpdateSettingsRequest	true	"Settings"
//	@Success		200		{object}	SettingsView
//	@Failure		422		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/settings [put]
func (a *API) updateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if !a.BindJSON(c, &req) {
		return
	}
	spec, err := a.Settings.Update(c, req.PanelSettingsSpec)
	if err == nil && req.OIDCClientSecret != nil {
		err = a.Settings.UpdateOIDCClientSecret(c, strings.TrimSpace(*req.OIDCClientSecret))
	}
	if err != nil {
		a.Fail(c, err)
		return
	}
	a.Audit(c, "panel settings updated", "oidcClientSecret", req.OIDCClientSecret != nil)
	a.sendSettings(c, spec)
}

// LegalTexts are the imprint and privacy policy shown in the footer of every page.
type LegalTexts struct {
	LegalNotice   string `json:"legalNotice"`
	PrivacyPolicy string `json:"privacyPolicy"`
}

// getLegalTexts godoc
//
//	@Summary		Legal notice and privacy policy
//	@Description	Public (also shown on the sign-in page). Markdown; empty when not configured.
//	@Description	Sends an ETag: with If-None-Match it answers 304 while nothing changed.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	LegalTexts
//	@Router			/legal [get]
func (a *API) getLegalTexts(c *gin.Context) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		a.Fail(c, err)
		return
	}
	a.revalidatedJSON(c, LegalTexts{LegalNotice: spec.LegalNotice, PrivacyPolicy: spec.PrivacyPolicy})
}

// Branding is how the panel presents itself: name and tagline (by default app.Name and app.BrandTagline) and
// the images as data URLs (empty for the built-in logo and icon).
type Branding struct {
	Name    string `json:"name"              example:"Acme Panel"`
	Tagline string `json:"tagline"           example:"Run by the Acme team"`
	Logo    string `json:"logo,omitempty"`
	Favicon string `json:"favicon,omitempty"`
}

// getBranding godoc
//
//	@Summary		Name, tagline, logo and favicon of the panel
//	@Description	Public (also shown on the sign-in page). The footer always names the software.
//	@Description	Sends an ETag: with If-None-Match it answers 304 while nothing changed.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	Branding
//	@Router			/branding [get]
func (a *API) getBranding(c *gin.Context) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		a.Fail(c, err)
		return
	}
	a.revalidatedJSON(c, Branding{
		Name:    cmp.Or(spec.BrandName, app.Name),
		Tagline: cmp.Or(spec.BrandTagline, app.BrandTagline),
		Logo:    spec.BrandLogo,
		Favicon: spec.Favicon,
	})
}

// revalidatedJSON answers public texts and images with a weak ETag: the browser keeps the answer and asks with
// If-None-Match, so logo and favicon (up to 128 KiB each) are sent again only after they changed (304 else).
func (a *API) revalidatedJSON(c *gin.Context, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		a.Fail(c, err)
		return
	}
	sum := sha256.Sum256(body)
	etag := `W/"` + hex.EncodeToString(sum[:16]) + `"`
	c.Header("Cache-Control", "no-cache")
	c.Header("ETag", etag)
	if strings.Contains(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}
