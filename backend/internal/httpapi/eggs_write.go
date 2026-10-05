package httpapi

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/eggstore"
)

// ImportURLRequest imports an egg from a URL.
type ImportURLRequest struct {
	URL string `json:"url" binding:"required" example:"https://raw.githubusercontent.com/pelican-eggs/minecraft/main/java/paper/egg-paper.json"`
	// AutoUpdate turns the hourly update from the URL on (an egg imported again keeps it on).
	AutoUpdate bool `json:"autoUpdate,omitempty"`
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
//	@Failure		422		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/eggs/import [post]
func (a *API) importEgg(c *gin.Context) {
	var data []byte
	var err error
	if fh, ferr := c.FormFile("file"); ferr == nil {
		f, oerr := fh.Open()
		if oerr != nil {
			a.fail(c, badRequest(oerr))
			return
		}
		defer f.Close()
		data, err = io.ReadAll(io.LimitReader(f, eggstore.MaxSize))
	} else {
		data, err = io.ReadAll(io.LimitReader(c.Request.Body, eggstore.MaxSize))
	}
	if err != nil {
		a.fail(c, badRequest(err))
		return
	}
	a.saveImport(c, data, "", false)
}

// importEggURL godoc
//
//	@Summary	Import an egg from a URL
//	@Tags		Eggs
//	@Accept		json
//	@Produce	json
//	@Param		body	body		ImportURLRequest	true	"Egg URL"
//	@Success	200		{object}	v1alpha1.Egg
//	@Failure	422		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/eggs/import-url [post]
func (a *API) importEggURL(c *gin.Context) {
	var req ImportURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	data, err := eggstore.Download(c, req.URL)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.saveImport(c, data, req.URL, req.AutoUpdate)
}

// saveImport stores an imported egg file and answers with the egg.
func (a *API) saveImport(c *gin.Context, data []byte, source string, autoUpdate bool) {
	e, created, err := a.Eggs.Import(c, data, source, autoUpdate)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "egg imported", "egg", e.Name, "source", source)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, e)
}

// deleteEgg godoc
//
//	@Summary	Delete an egg
//	@Tags		Eggs
//	@Param		egg	path	string	true	"Egg name"
//	@Success	204
//	@Failure	409	{object}	ErrorResponse	"egg is used by servers"
//	@Security	BearerAuth
//	@Router		/eggs/{egg} [delete]
func (a *API) deleteEgg(c *gin.Context) {
	if err := a.Eggs.Delete(c, c.Param("egg")); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "egg deleted", "egg", c.Param("egg"))
	c.Status(http.StatusNoContent)
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
//	@Failure		422		{object}	ErrorResponse	"validation errors per field"
//	@Security		BearerAuth
//	@Router			/eggs [post]
func (a *API) createEgg(c *gin.Context) {
	spec := &v1alpha1.EggSpec{}
	if err := c.ShouldBindJSON(spec); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	e, err := a.Eggs.Create(c, spec)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "egg created", "egg", e.Name)
	c.JSON(http.StatusCreated, e)
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
//	@Failure		422		{object}	ErrorResponse	"validation errors per field"
//	@Security		BearerAuth
//	@Router			/eggs/{egg} [put]
func (a *API) updateEgg(c *gin.Context) {
	spec := &v1alpha1.EggSpec{}
	if err := c.ShouldBindJSON(spec); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	e, err := a.Eggs.Update(c, c.Param("egg"), spec)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "egg updated", "egg", e.Name)
	c.JSON(http.StatusOK, e)
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
//	@Failure		409	{object}	ErrorResponse	"the egg has no update URL"
//	@Security		BearerAuth
//	@Router			/eggs/{egg}/update-from-url [post]
func (a *API) updateEggFromURL(c *gin.Context) {
	e, err := a.Eggs.UpdateFromURL(c, c.Param("egg"))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "egg updated from its URL", "egg", e.Name, "url", e.Spec.Source.UpdateURL)
	c.JSON(http.StatusOK, e)
}
