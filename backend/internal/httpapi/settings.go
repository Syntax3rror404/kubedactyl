package httpapi

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/kube"
	"app/internal/settings"
)

// StorageClassList lists the storage classes of the cluster.
type StorageClassList struct {
	Items []settings.StorageClass `json:"items"`
}

// PoolList lists the load balancer IP pools of the cluster.
type PoolList struct {
	Items []settings.Pool `json:"items"`
	// Error is set when the pools cannot be read (e.g. Cilium LB IPAM is not installed).
	Error string `json:"error,omitempty"`
}

// RequestRates are the requests per second of the signed-in user and, for administrators, the
// Kubernetes API calls of the whole panel, each with its limit (settings "Kube API limit").
type RequestRates struct {
	User  kube.Rate  `json:"user"`
	Panel *kube.Rate `json:"panel,omitempty" extensions:"x-nullable"`
}

// getRequestRates godoc
//
//	@Summary		Request rates
//	@Description	Averages over the last 5 seconds; refused requests and waiting calls count, too. This request
//	@Description	does not count. Panel is set for administrators only.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	RequestRates
//	@Security		BearerAuth
//	@Router			/request-rates [get]
func (a *API) getRequestRates(c *gin.Context) {
	var out RequestRates
	if a.KubeLimiter != nil {
		p := principal(c)
		out.User = a.KubeLimiter.UserRate(p.User.Name)
		if p.Admin() {
			panel := a.KubeLimiter.PanelRate()
			out.Panel = &panel
		}
	}
	c.JSON(http.StatusOK, out)
}

// SettingsView is the panel settings; the client secret of the identity provider is never sent.
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

// settingsView adds whether a client secret is stored, for administrators (only they edit the settings).
func (a *API) settingsView(c *gin.Context, spec v1alpha1.PanelSettingsSpec) (SettingsView, error) {
	if !principal(c).Admin() {
		return SettingsView{PanelSettingsSpec: spec}, nil
	}
	secret, err := a.Settings.OIDCClientSecret(c)
	return SettingsView{PanelSettingsSpec: spec, OIDCClientSecretSet: secret != ""}, err
}

// getSettings godoc
//
//	@Summary		Panel settings
//	@Description	External domain and the storage classes and load balancer pools that can be selected for servers.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	SettingsView
//	@Security		BearerAuth
//	@Router			/settings [get]
func (a *API) getSettings(c *gin.Context) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	view, err := a.settingsView(c, spec)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

// updateSettings godoc
//
//	@Summary		Update the panel settings
//	@Description	Every storage class and pool must exist in the cluster; pools must be selectable by service
//	@Description	labels. Missing defaults become the first entry.
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
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	spec, err := a.Settings.Update(c, req.PanelSettingsSpec)
	if err == nil && req.OIDCClientSecret != nil {
		err = a.Settings.UpdateOIDCClientSecret(c, strings.TrimSpace(*req.OIDCClientSecret))
	}
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "panel settings updated", "oidcClientSecret", req.OIDCClientSecret != nil)
	view, err := a.settingsView(c, spec)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

// listStorageClasses godoc
//
//	@Summary	Storage classes of the cluster
//	@Tags		Settings
//	@Produce	json
//	@Success	200	{object}	StorageClassList
//	@Security	BearerAuth
//	@Router		/settings/storage-classes [get]
func (a *API) listStorageClasses(c *gin.Context) {
	classes, err := settings.ListStorageClasses(c, a.Reader)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, StorageClassList{Items: classes})
}

// listLoadBalancerPools godoc
//
//	@Summary		Load balancer IP pools of the cluster
//	@Description	Cilium LB IPAM pools with their address blocks, usage and the service labels that select them.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	PoolList
//	@Security		BearerAuth
//	@Router			/settings/load-balancer-pools [get]
func (a *API) listLoadBalancerPools(c *gin.Context) {
	pools, err := settings.ListPools(c, a.Reader)
	if errors.Is(err, settings.ErrNoCilium) {
		c.JSON(http.StatusOK, PoolList{Items: []settings.Pool{}, Error: err.Error()})
		return
	}
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, PoolList{Items: pools})
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
		a.fail(c, err)
		return
	}
	a.revalidatedJSON(c, LegalTexts{LegalNotice: spec.LegalNotice, PrivacyPolicy: spec.PrivacyPolicy})
}

// Branding is how the panel presents itself: name and tagline (defaults: Kubedactyl, "Game
// servers on Kubernetes") and the images as data URLs (empty: the built-in logo and icon).
type Branding struct {
	Name    string `json:"name"              example:"Kubedactyl"`
	Tagline string `json:"tagline"           example:"Game servers on Kubernetes"`
	Logo    string `json:"logo,omitempty"`
	Favicon string `json:"favicon,omitempty"`
}

// getBranding godoc
//
//	@Summary		Name, tagline, logo and favicon of the panel
//	@Description	Public (also shown on the sign-in page). The footer always names the software (Kubedactyl).
//	@Description	Sends an ETag: with If-None-Match it answers 304 while nothing changed.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	Branding
//	@Router			/branding [get]
func (a *API) getBranding(c *gin.Context) {
	spec, err := a.Settings.Current(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.revalidatedJSON(c, Branding{
		Name:    cmp.Or(spec.BrandName, "Kubedactyl"),
		Tagline: cmp.Or(spec.BrandTagline, "Game servers on Kubernetes"),
		Logo:    spec.BrandLogo,
		Favicon: spec.Favicon,
	})
}

// revalidatedJSON answers public texts and images with a weak ETag: the browser keeps the answer and asks with
// If-None-Match, so logo and favicon (up to 128 KiB each) are sent again only after they changed (304 else).
func (a *API) revalidatedJSON(c *gin.Context, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		a.fail(c, err)
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
