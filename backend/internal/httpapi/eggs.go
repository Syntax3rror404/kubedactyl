package httpapi

import (
	"bytes"
	"cmp"
	"net/http"
	"sort"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/base/api"
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
func (a *API) listEggs(w http.ResponseWriter, r *http.Request) {
	var list v1alpha1.EggList
	if err := a.Client.List(r.Context(), &list, client.InNamespace(a.Opts.Namespace)); err != nil {
		a.Fail(w, r, err)
		return
	}
	sort.Slice(
		list.Items,
		func(i, j int) bool { return list.Items[i].Spec.DisplayName < list.Items[j].Spec.DisplayName },
	)
	p := api.PrincipalOf(r)
	for i := range list.Items {
		visibleEgg(p, &list.Items[i])
	}
	api.JSON(w, http.StatusOK, EggList{Items: list.Items})
}

// getEgg godoc
//
//	@Summary	Get an egg
//	@Tags		Eggs
//	@Produce	json
//	@Param		egg	path		string	true	"Egg name"
//	@Success	200	{object}	v1alpha1.Egg
//	@Failure	404	{object}	api.ErrorResponse
//	@Security	BearerAuth
//	@Router		/eggs/{egg} [get]
func (a *API) getEgg(w http.ResponseWriter, r *http.Request) {
	if e, ok := a.loadEgg(w, r, r.PathValue("egg")); ok {
		api.JSON(w, http.StatusOK, visibleEgg(api.PrincipalOf(r), e))
	}
}

// loadEgg reads an egg (just created ones, too: eggstore.Store.Get).
func (a *API) loadEgg(w http.ResponseWriter, r *http.Request, name string) (*v1alpha1.Egg, bool) {
	e, err := a.Eggs.Get(r.Context(), name)
	if err != nil {
		a.Fail(w, r, err)
		return nil, false
	}
	return e, true
}

// eggsByName returns all eggs keyed by name (for visibleServer); empty when they cannot be read.
func (a *API) eggsByName(w http.ResponseWriter, r *http.Request) map[string]*v1alpha1.Egg {
	var list v1alpha1.EggList
	out := map[string]*v1alpha1.Egg{}
	if err := a.Client.List(r.Context(), &list, client.InNamespace(a.Opts.Namespace)); err == nil {
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
func (a *API) exportEgg(w http.ResponseWriter, r *http.Request) {
	e, ok := a.loadEgg(w, r, r.PathValue("egg"))
	if !ok {
		return
	}
	format := cmp.Or(r.URL.Query().Get("format"), egg.FormatYAML)
	data, file, err := egg.Export(e.Name, &e.Spec, format, time.Now())
	if err != nil {
		a.Fail(w, r, api.BadRequest(err))
		return
	}
	if r.URL.Query().Get("download") == "true" {
		_ = a.Download(w, r, file, api.Copy(bytes.NewReader(data)))
		return
	}
	ctype := "application/json; charset=utf-8"
	if format == egg.FormatYAML {
		ctype = "application/yaml; charset=utf-8"
	}
	api.Data(w, http.StatusOK, ctype, data)
}
