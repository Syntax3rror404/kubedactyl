package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/egglibrary"
	"app/internal/settings"
	"app/internal/validation"
)

// LibraryList holds the eggs of every repository configured as egg library.
type LibraryList struct {
	Repositories []egglibrary.Repository `json:"repositories"`
}

// LibraryRepositoryCheck tells whether a repository can be read and how many eggs it holds.
type LibraryRepositoryCheck struct {
	// URL is the normalized repository URL.
	URL  string `json:"url"`
	Eggs int    `json:"eggs"`
	// Error tells why the repository could not be read.
	Error string `json:"error,omitempty"`
}

// LibraryEgg is an egg file of the library with its content.
type LibraryEgg struct {
	Egg  egglibrary.Egg   `json:"egg"`
	Spec v1alpha1.EggSpec `json:"spec"`
}

// listLibraryEggs godoc
//
//	@Summary		List the eggs of the egg library
//	@Description	Reads the git repositories set in the settings (eggLibraries). Each is kept in the panel's
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

// getLibraryRepository godoc
//
//	@Summary		Check a repository of the egg library
//	@Description	Reads a git repository whether it is configured or not, so the settings can check it
//	@Description	before it is saved. Its eggs are kept in the panel's memory like a listed repository.
//	@Tags			Eggs
//	@Produce		json
//	@Param			url	query		string	true	"repository URL (https://<host>/<owner>/<repo>)"
//	@Success		200	{object}	LibraryRepositoryCheck
//	@Failure		422	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/egg-library/repository [get]
func (a *API) getLibraryRepository(c *gin.Context) {
	url, err := settings.NormalizeEggLibrary(c.Query("url"))
	if err == nil && url == "" {
		err = errors.New("enter a repository URL")
	}
	if err != nil {
		a.fail(c, validation.Field("url", err))
		return
	}
	r := a.Library.GetRepository(c, url)
	c.JSON(http.StatusOK, LibraryRepositoryCheck{URL: r.URL, Eggs: len(r.Eggs), Error: r.Error})
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
