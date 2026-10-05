package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/egglibrary"
)

// LibraryList holds the eggs of every repository configured as egg library.
type LibraryList struct {
	Repositories []egglibrary.Repository `json:"repositories"`
}

// LibraryEgg is an egg file of the library with its content.
type LibraryEgg struct {
	Egg  egglibrary.Egg   `json:"egg"`
	Spec v1alpha1.EggSpec `json:"spec"`
}

// listLibraryEggs godoc
//
//	@Summary		List the eggs of the egg library
//	@Description	Reads the GitHub repositories set in the settings (eggLibraries). Each is kept in the panel's
//	@Description	memory for 15 minutes; refresh=true downloads them again.
//	@Tags			Eggs
//	@Produce		json
//	@Param			refresh	query		bool	false	"download the repositories again"
//	@Success		200		{object}	LibraryList
//	@Security		BearerAuth
//	@Router			/egg-library [get]
func (a *API) listLibraryEggs(c *gin.Context) {
	set, err := a.Settings.Get(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	repos := a.Library.List(c, set.EggLibraries, c.Query("refresh") == "true")
	c.JSON(http.StatusOK, LibraryList{Repositories: repos})
}

// getLibraryEgg godoc
//
//	@Summary	Get an egg of the egg library
//	@Tags		Eggs
//	@Produce	json
//	@Param		repository	query		string	true	"repository URL"
//	@Param		path		query		string	true	"path of the egg file in the repository"
//	@Success	200			{object}	LibraryEgg
//	@Failure	404			{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/egg-library/egg [get]
func (a *API) getLibraryEgg(c *gin.Context) {
	set, err := a.Settings.Get(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	spec, e, err := a.Library.Get(c, set.EggLibraries, c.Query("repository"), c.Query("path"))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, LibraryEgg{Egg: e, Spec: *spec})
}
