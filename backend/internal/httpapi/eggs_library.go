package httpapi

import (
	"errors"
	"net/http"

	"app/api/v1alpha1"
	"app/base/api"
	"app/base/validation"
	"app/internal/appsettings"
	"app/internal/egglibrary"
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
func (a *API) listLibraryEggs(w http.ResponseWriter, r *http.Request) {
	set, err := a.Settings.Current(r.Context())
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	repos := a.Library.List(r.Context(), set.EggLibraries, r.URL.Query().Get("refresh") == "true")
	api.JSON(w, http.StatusOK, LibraryList{Repositories: repos})
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
//	@Failure		422	{object}	api.ErrorResponse
//	@Security		BearerAuth
//	@Router			/egg-library/repository [get]
func (a *API) getLibraryRepository(w http.ResponseWriter, r *http.Request) {
	url, err := appsettings.NormalizeEggLibrary(r.URL.Query().Get("url"))
	if err == nil && url == "" {
		err = errors.New("enter a repository URL")
	}
	if err != nil {
		a.Fail(w, r, validation.Field("url", err))
		return
	}
	repo := a.Library.GetRepository(r.Context(), url)
	api.JSON(w, http.StatusOK, LibraryRepositoryCheck{URL: repo.URL, Eggs: len(repo.Eggs), Error: repo.Error})
}

// getLibraryEgg godoc
//
//	@Summary	Get an egg of the egg library
//	@Tags		Eggs
//	@Produce	json
//	@Param		repository	query		string	true	"repository URL"
//	@Param		path		query		string	true	"path of the egg file in the repository"
//	@Success	200			{object}	LibraryEgg
//	@Failure	404			{object}	api.ErrorResponse
//	@Security	BearerAuth
//	@Router		/egg-library/egg [get]
func (a *API) getLibraryEgg(w http.ResponseWriter, r *http.Request) {
	set, err := a.Settings.Current(r.Context())
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	query := r.URL.Query()
	spec, e, err := a.Library.Get(r.Context(), set.EggLibraries, query.Get("repository"), query.Get("path"))
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	api.JSON(w, http.StatusOK, LibraryEgg{Egg: e, Spec: *spec})
}
