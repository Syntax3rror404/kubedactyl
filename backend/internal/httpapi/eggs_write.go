package httpapi

import (
	"errors"
	"net/http"

	"app/api/v1alpha1"
	"app/base/api"
	"app/base/validation"
	"app/internal/eggstore"
)

// ImportURLRequest imports an egg from a URL.
type ImportURLRequest struct {
	URL string `json:"url" example:"https://raw.githubusercontent.com/pelican-eggs/minecraft/main/java/paper/egg-paper.json"`
	// AutoUpdate turns the hourly update from the URL on (an egg imported again keeps it on).
	AutoUpdate bool `json:"autoUpdate,omitempty"`
}

// Validate checks that a URL is given.
func (req ImportURLRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("url", req.URL != "")
	return errs.OrNil()
}

// importEgg godoc
//
//	@Summary		Import an egg file
//	@Description	Accepts a Pterodactyl (PTDL_v1/v2) or Pelican (PLCN_v1-v3) egg as JSON or YAML, either as request
//	@Description	body or as multipart field "file". An egg with the same name is updated.
//	@Tags			Eggs
//	@Accept			json,mpfd
//	@Produce		json
//	@Param			file	formData	file			false	"Egg file"
//	@Success		200		{object}	v1alpha1.Egg	"an egg with the same name was updated"
//	@Success		201		{object}	v1alpha1.Egg	"a new egg was created"
//	@Failure		422		{object}	api.ErrorResponse
//	@Security		BearerAuth
//	@Router			/eggs/import [post]
func (a *API) importEgg(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var err error
	if f, _, ferr := r.FormFile("file"); ferr == nil {
		defer f.Close()
		data, err = eggstore.ReadFile(f)
	} else {
		data, err = eggstore.ReadFile(r.Body)
	}
	if err != nil && !errors.Is(err, eggstore.ErrTooLarge) {
		err = api.BadRequest(err)
	}
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.saveImport(w, r, data, "", false)
}

// importEggURL godoc
//
//	@Summary	Import an egg from a URL
//	@Tags		Eggs
//	@Accept		json
//	@Produce	json
//	@Param		body	body		ImportURLRequest	true	"Egg URL"
//	@Success	200		{object}	v1alpha1.Egg		"an egg with the same name was updated"
//	@Success	201		{object}	v1alpha1.Egg		"a new egg was created"
//	@Failure	422		{object}	api.ErrorResponse
//	@Security	BearerAuth
//	@Router		/eggs/import-url [post]
func (a *API) importEggURL(w http.ResponseWriter, r *http.Request) {
	var req ImportURLRequest
	if !a.Decode(w, r, &req) {
		return
	}
	data, err := eggstore.Download(r.Context(), req.URL)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.saveImport(w, r, data, req.URL, req.AutoUpdate)
}

// saveImport stores an imported egg file and answers with the egg.
func (a *API) saveImport(w http.ResponseWriter, r *http.Request, data []byte, source string, autoUpdate bool) {
	e, created, err := a.Eggs.Import(r.Context(), data, source, autoUpdate)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "egg imported", "egg", e.Name, "source", source)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	api.JSON(w, status, e)
}

// deleteEgg godoc
//
//	@Summary	Delete an egg
//	@Tags		Eggs
//	@Param		egg	path	string	true	"Egg name"
//	@Success	204
//	@Failure	409	{object}	api.ErrorResponse	"egg is used by servers"
//	@Security	BearerAuth
//	@Router		/eggs/{egg} [delete]
func (a *API) deleteEgg(w http.ResponseWriter, r *http.Request) {
	if err := a.Eggs.Delete(r.Context(), r.PathValue("egg")); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "egg deleted", "egg", r.PathValue("egg"))
	w.WriteHeader(http.StatusNoContent)
}

// createEgg godoc
//
//	@Summary		Create an egg
//	@Description	Creates an egg from the fields of the editor (also used to duplicate one). The object name is
//	@Description	derived from the display name; a new UUID is assigned.
//	@Tags			Eggs
//	@Accept			json
//	@Produce		json
//	@Param			body	body		v1alpha1.EggSpec	true	"Egg"
//	@Success		201		{object}	v1alpha1.Egg
//	@Failure		422		{object}	api.ErrorResponse	"validation errors per field"
//	@Security		BearerAuth
//	@Router			/eggs [post]
func (a *API) createEgg(w http.ResponseWriter, r *http.Request) {
	spec := &v1alpha1.EggSpec{}
	if !a.Decode(w, r, spec) {
		return
	}
	e, err := a.Eggs.Create(r.Context(), spec)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "egg created", "egg", e.Name)
	api.JSON(w, http.StatusCreated, e)
}

// updateEgg godoc
//
//	@Summary		Update an egg
//	@Description	Replaces the fields of an egg. Servers use the changes from their next start (install script: next
//	@Description	reinstall). UUID and import details are kept.
//	@Tags			Eggs
//	@Accept			json
//	@Produce		json
//	@Param			egg		path		string				true	"Egg name"
//	@Param			body	body		v1alpha1.EggSpec	true	"Egg"
//	@Success		200		{object}	v1alpha1.Egg
//	@Failure		422		{object}	api.ErrorResponse	"validation errors per field"
//	@Security		BearerAuth
//	@Router			/eggs/{egg} [put]
func (a *API) updateEgg(w http.ResponseWriter, r *http.Request) {
	spec := &v1alpha1.EggSpec{}
	if !a.Decode(w, r, spec) {
		return
	}
	e, err := a.Eggs.Update(r.Context(), r.PathValue("egg"), spec)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "egg updated", "egg", e.Name)
	api.JSON(w, http.StatusOK, e)
}

// updateEggFromURL godoc
//
//	@Summary		Update an egg from its update URL
//	@Description	Downloads the egg from meta.update_url and replaces all fields, including changes made in the
//	@Description	panel. Name and UUID are kept.
//	@Tags			Eggs
//	@Produce		json
//	@Param			egg	path		string	true	"Egg name"
//	@Success		200	{object}	v1alpha1.Egg
//	@Failure		409	{object}	api.ErrorResponse	"the egg has no update URL"
//	@Security		BearerAuth
//	@Router			/eggs/{egg}/update-from-url [post]
func (a *API) updateEggFromURL(w http.ResponseWriter, r *http.Request) {
	e, err := a.Eggs.UpdateFromURL(r.Context(), r.PathValue("egg"))
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "egg updated from its URL", "egg", e.Name, "url", e.Spec.Source.UpdateURL)
	api.JSON(w, http.StatusOK, e)
}
