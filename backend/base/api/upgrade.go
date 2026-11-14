package api

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"slices"
	"time"

	"github.com/gin-gonic/gin"

	"app/app"
	"app/base/selfupgrade"
)

// UpgradeStatus describes panel updates: the installed version, newer chart versions in the
// registry and the upgrade jobs.
type UpgradeStatus struct {
	// Enabled is false without the chart value selfUpgrade.enabled.
	Enabled bool   `json:"enabled"`
	Current string `json:"current"          example:"0.2.3"`
	Chart   string `json:"chart,omitempty"  example:"oci://ghcr.io/example/charts/panel"`
	Latest  string `json:"latest,omitempty" example:"0.2.4"`
	// Newer lists the versions above the current one, highest first.
	Newer     []string          `json:"newer"`
	CheckedAt *time.Time        `json:"checkedAt,omitempty"`
	Error     string            `json:"error,omitempty"`
	Jobs      []selfupgrade.Job `json:"jobs"`
	// IntervalSeconds is how often the registry is checked.
	IntervalSeconds int `json:"intervalSeconds" example:"600"`
}

// StartUpgradeRequest selects the chart version to upgrade to.
type StartUpgradeRequest struct {
	Version string `json:"version" binding:"required" example:"0.2.4"`
}

var errUpgradesDisabled = errors.New("self-upgrades are not enabled (chart value selfUpgrade.enabled)")

// getUpgradeStatus godoc
//
//	@Summary		Panel updates
//	@Description	Newer chart versions (checked every 10 minutes) and the upgrade jobs. refresh=true checks the
//	@Description	registry now.
//	@Tags			System
//	@Produce		json
//	@Param			refresh	query		bool	false	"Check the registry now"
//	@Success		200		{object}	UpgradeStatus
//	@Security		BearerAuth
//	@Router			/upgrade [get]
func (a *API) getUpgradeStatus(c *gin.Context) {
	st := UpgradeStatus{
		Current:         a.Version,
		Newer:           []string{},
		Jobs:            []selfupgrade.Job{},
		IntervalSeconds: int(selfupgrade.CheckInterval.Seconds()),
	}
	if a.Upgrader == nil {
		c.JSON(http.StatusOK, st)
		return
	}
	st.Enabled, st.Chart = true, a.Upgrader.Config.Chart
	check := a.UpgradeChecker.Last()
	if c.Query("refresh") == "true" {
		check = a.UpgradeChecker.Refresh()
	}
	if !check.CheckedAt.IsZero() {
		st.CheckedAt = &check.CheckedAt
	}
	st.Latest, st.Error = check.Latest, check.Error
	if check.Newer != nil {
		st.Newer = check.Newer
	}
	jobs, err := a.Upgrader.Jobs(c, true)
	if err != nil {
		a.Fail(c, err)
		return
	}
	st.Jobs = jobs
	c.JSON(http.StatusOK, st)
}

// startUpgrade godoc
//
//	@Summary		Upgrade the panel
//	@Description	Starts a job that runs helm upgrade with the release's values and the chosen chart version. The
//	@Description	panel restarts on the new version.
//	@Tags			System
//	@Accept			json
//	@Produce		json
//	@Param			body	body		StartUpgradeRequest	true	"Version"
//	@Success		202		{object}	selfupgrade.Job
//	@Failure		409		{object}	ErrorResponse	"disabled, unknown version or an upgrade is running"
//	@Security		BearerAuth
//	@Router			/upgrade [post]
func (a *API) startUpgrade(c *gin.Context) {
	var req StartUpgradeRequest
	if !a.BindJSON(c, &req) {
		return
	}
	if a.Upgrader == nil {
		a.Fail(c, Conflict(errUpgradesDisabled))
		return
	}
	// Only versions the registry offers above the current one (no downgrades).
	if !slices.Contains(a.UpgradeChecker.Last().Newer, req.Version) &&
		!slices.Contains(a.UpgradeChecker.Refresh().Newer, req.Version) {
		a.Fail(c, Conflict(fmt.Errorf("version %q is not a newer version in %s", req.Version, a.Upgrader.Config.Chart)))
		return
	}
	job, err := a.Upgrader.Start(c, req.Version)
	if err != nil {
		a.Fail(c, err)
		return
	}
	a.Audit(c, "panel upgrade started", "from", a.Version, "to", req.Version, "job", job.Name)
	c.JSON(http.StatusAccepted, job)
}

// Versions are the versions of the Go runtime and the main Go modules of the panel.
type Versions struct {
	Go      string          `json:"go"      example:"go1.27.1"`
	Modules []ModuleVersion `json:"modules"`
}

// ModuleVersion is a Go module built into the panel.
type ModuleVersion struct {
	Name    string `json:"name"    example:"Gin"`
	Version string `json:"version" example:"v1.12.0"`
}

// versionModules are the Go modules shown with their version (label and module path): the base ones, then the
// app's.
var versionModules = append([][2]string{
	{"Gin", "github.com/gin-gonic/gin"},
	{"controller-runtime", "sigs.k8s.io/controller-runtime"},
	{"client-go", "k8s.io/client-go"},
	{"Helm", "helm.sh/helm/v4"},
}, app.VersionModules...)

// getVersions godoc
//
//	@Summary	Software versions
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	Versions
//	@Security	BearerAuth
//	@Router		/versions [get]
func (a *API) getVersions(c *gin.Context) {
	out := Versions{Go: runtime.Version(), Modules: []ModuleVersion{}}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, m := range versionModules {
			for _, dep := range info.Deps {
				if dep.Path == m[1] {
					out.Modules = append(out.Modules, ModuleVersion{Name: m[0], Version: dep.Version})
				}
			}
		}
	}
	c.JSON(http.StatusOK, out)
}
