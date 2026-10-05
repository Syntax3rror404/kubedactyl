package httpapi

import (
	"net/http"
	"runtime"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Versions are the versions of the panel, its Go runtime and its main Go modules.
type Versions struct {
	Panel   string          `json:"panel"   example:"0.2.37"`
	Go      string          `json:"go"      example:"go1.27.1"`
	Modules []ModuleVersion `json:"modules"`
}

// ModuleVersion is a Go module built into the panel.
type ModuleVersion struct {
	Name    string `json:"name"    example:"Gin"`
	Version string `json:"version" example:"v1.12.0"`
}

// versionModules are the Go modules shown with their version.
var versionModules = []struct{ name, path string }{
	{"Gin", "github.com/gin-gonic/gin"},
	{"controller-runtime", "sigs.k8s.io/controller-runtime"},
	{"client-go", "k8s.io/client-go"},
	{"Helm", "helm.sh/helm/v4"},
}

// getVersions godoc
//
//	@Summary	Software versions
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	Versions
//	@Security	BearerAuth
//	@Router		/versions [get]
func (a *API) getVersions(c *gin.Context) {
	out := Versions{Panel: a.Version, Go: runtime.Version(), Modules: []ModuleVersion{}}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, m := range versionModules {
			for _, dep := range info.Deps {
				if dep.Path == m.path {
					out.Modules = append(out.Modules, ModuleVersion{Name: m.name, Version: dep.Version})
				}
			}
		}
	}
	c.JSON(http.StatusOK, out)
}
