package httpapi

import (
	"errors"
	"net/http"

	"app/internal/appsettings"
	"app/internal/appusers"
	"app/internal/egglibrary"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/schedule"
	"app/internal/serverctl"
)

// StatusOf is the HTTP status of an error of the app's domain packages (0: none of them); the base API answers
// every failed request (api.API.Fail).
func (a *API) StatusOf(err error) int {
	for _, m := range domainErrors {
		if errors.Is(err, m.err) {
			return m.status
		}
	}
	return 0
}

// domainErrors are the errors of the app's domain packages and their HTTP status (the base maps its own).
var domainErrors = []struct {
	err    error
	status int
}{
	{egglibrary.ErrNotFound, http.StatusNotFound},
	{eggstore.ErrInvalid, http.StatusUnprocessableEntity},
	{eggstore.ErrTooLarge, http.StatusUnprocessableEntity},
	{eggstore.ErrInvalidURL, http.StatusUnprocessableEntity},
	{eggstore.ErrNoUpdateURL, http.StatusConflict},
	{eggstore.ErrDownload, http.StatusBadGateway},
	{eggstore.ErrInUse, http.StatusConflict},
	{eggstore.ErrNameTaken, http.StatusConflict},
	{files.ErrNotFound, http.StatusNotFound},
	{files.ErrExists, http.StatusConflict},
	{files.ErrDenied, http.StatusForbidden},
	{files.ErrTooLarge, http.StatusRequestEntityTooLarge},
	{files.ErrTooManyEntries, http.StatusRequestEntityTooLarge},
	{files.ErrForeignPod, http.StatusConflict},
	{files.ErrBusy, http.StatusConflict},
	{files.ErrJobNotFound, http.StatusNotFound},
	{files.ErrRestoring, http.StatusConflict},
	{files.ErrBadURL, http.StatusUnprocessableEntity},
	{files.ErrBadName, http.StatusUnprocessableEntity},
	{files.ErrNameRequired, http.StatusUnprocessableEntity},
	{files.ErrIntoItself, http.StatusUnprocessableEntity},
	{schedule.ErrInvalid, http.StatusUnprocessableEntity},
	{schedule.ErrNotFound, http.StatusNotFound},
	{schedule.ErrRunning, http.StatusConflict},
	{serverctl.ErrOffline, http.StatusConflict},
	{serverctl.ErrRunning, http.StatusConflict},
	{serverctl.ErrSuspended, http.StatusForbidden},
	{serverctl.ErrRemoving, http.StatusConflict},
	{serverctl.ErrSameOwner, http.StatusConflict},
	{serverctl.ErrFilesBusy, http.StatusConflict},
	{serverctl.ErrNoVolume, http.StatusConflict},
	{serverctl.ErrMigrating, http.StatusConflict},
	{serverctl.ErrSameStorageClass, http.StatusConflict},
	{serverctl.ErrInstalling, http.StatusConflict},
	{serverctl.ErrNotMigrating, http.StatusConflict},
	{serverctl.ErrSwitching, http.StatusConflict},
	{appsettings.ErrNotConfigured, http.StatusUnprocessableEntity},
	{appsettings.ErrNotEnabled, http.StatusUnprocessableEntity},
	{appsettings.ErrNoCilium, http.StatusUnprocessableEntity},
	{appusers.ErrSubusersOff, http.StatusForbidden},
	{appusers.ErrSubuserLimit, http.StatusConflict},
	{appusers.ErrSubuser, http.StatusUnprocessableEntity},
}
