package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/egg"
)

// EggList is a list of eggs.
type EggList struct {
	Items []v1alpha1.Egg `json:"items"`
}

// listEggs godoc
//
//	@Summary	List eggs
//	@Tags		Eggs
//	@Produce	json
//	@Success	200	{object}	EggList
//	@Security	BearerAuth
//	@Router		/eggs [get]
func (a *API) listEggs(c *gin.Context) {
	var list v1alpha1.EggList
	if err := a.Client.List(c, &list, client.InNamespace(a.Opts.Namespace)); err != nil {
		a.fail(c, err)
		return
	}
	sort.Slice(
		list.Items,
		func(i, j int) bool { return list.Items[i].Spec.DisplayName < list.Items[j].Spec.DisplayName },
	)
	for i := range list.Items {
		list.Items[i] = *visibleEgg(principal(c), &list.Items[i])
	}
	c.JSON(http.StatusOK, EggList{Items: list.Items})
}

// getEgg godoc
//
//	@Summary	Get an egg
//	@Tags		Eggs
//	@Produce	json
//	@Param		egg	path		string	true	"Egg name"
//	@Success	200	{object}	v1alpha1.Egg
//	@Failure	404	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/eggs/{egg} [get]
func (a *API) getEgg(c *gin.Context) {
	if e, ok := a.loadEgg(c, c.Param("egg")); ok {
		c.JSON(http.StatusOK, visibleEgg(principal(c), e))
	}
}

// eggsByName returns all eggs keyed by name (for sanitizing server responses).
func (a *API) eggsByName(c *gin.Context) map[string]*v1alpha1.Egg {
	var list v1alpha1.EggList
	out := map[string]*v1alpha1.Egg{}
	if err := a.Client.List(c, &list, client.InNamespace(a.Opts.Namespace)); err == nil {
		for i := range list.Items {
			out[list.Items[i].Name] = &list.Items[i]
		}
	}
	return out
}

// exportEgg godoc
//
//	@Summary		Export an egg
//	@Description	yaml / json: Pelican format PLCN_v3 (same content); ptdl: Pterodactyl format PTDL_v2 (JSON).
//	@Description	download=true sends it as file.
//	@Tags			Eggs
//	@Produce		plain
//	@Param			egg			path		string	true	"Egg name"
//	@Param			format		query		string	false	"yaml (default), json or ptdl"
//	@Param			download	query		bool	false	"Send as attachment"
//	@Success		200			{string}	string	"egg file"
//	@Security		BearerAuth
//	@Router			/eggs/{egg}/export [get]
func (a *API) exportEgg(c *gin.Context) {
	e, ok := a.loadEgg(c, c.Param("egg"))
	if !ok {
		return
	}
	format := c.DefaultQuery("format", egg.FormatYAML)
	data, file, err := egg.Export(e.Name, &e.Spec, format, time.Now())
	if err != nil {
		a.fail(c, badRequest(err))
		return
	}
	ctype := "application/json; charset=utf-8"
	if format == egg.FormatYAML {
		ctype = "application/yaml; charset=utf-8"
	}
	if c.Query("download") == "true" {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file))
	}
	c.Data(http.StatusOK, ctype, data)
}
