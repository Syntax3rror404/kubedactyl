package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"app/internal/egglibrary"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/kube"
	"app/internal/schedule"
	"app/internal/selfupgrade"
	"app/internal/serverctl"
	"app/internal/settings"
	"app/internal/users"
	"app/internal/validation"
)

// Every failed request is answered the same way: a.fail(c, err).
//
// Handlers and helpers choose the status by wrapping the error (badRequest(err), forbidden(err), …).
// Errors without a status are mapped by statusOf: validation errors (validation.Error, 422 with the
// messages per field), Kubernetes errors and the errors of the domain packages (domainErrors).
// Anything else is an internal error (500) and logged.

// statusError is an error with the HTTP status the request is answered with.
type statusError struct {
	status int
	err    error
	// fields holds validation errors per form field or variable.
	fields map[string]string
}

func (e *statusError) Error() string { return e.err.Error() }
func (e *statusError) Unwrap() error { return e.err }

func httpError(status int, err error) error { return &statusError{status: status, err: err} }

func badRequest(err error) error      { return httpError(http.StatusBadRequest, err) }
func unauthorized(err error) error    { return httpError(http.StatusUnauthorized, err) }
func forbidden(err error) error       { return httpError(http.StatusForbidden, err) }
func notFound(err error) error        { return httpError(http.StatusNotFound, err) }
func conflict(err error) error        { return httpError(http.StatusConflict, err) }
func tooManyRequests(err error) error { return httpError(http.StatusTooManyRequests, err) }

// fieldError is an error of one form field; the frontend shows it next to the field.
func fieldError(status int, field string, err error) error {
	return &statusError{status: status, err: err, fields: map[string]string{field: err.Error()}}
}

// fail answers a failed request and aborts the handler chain.
func (a *API) fail(c *gin.Context, err error) {
	se := statusOf(err)
	msg := se.err.Error()
	if se.status == http.StatusInternalServerError {
		a.Log.Error("request failed", "path", c.FullPath(), "err", err)
		// Details (object names, cluster errors) only for administrators.
		if p := principal(c); p == nil || !p.Admin() {
			msg = "internal error: the administrator can find details in the panel log"
		}
	}
	c.AbortWithStatusJSON(se.status, ErrorResponse{Error: msg, Fields: se.fields})
}

// statusOf finds the HTTP status of an error.
func statusOf(err error) *statusError {
	var se *statusError
	var ve validation.Error
	switch {
	case errors.As(err, &se):
		if se.fields == nil && errors.As(se.err, &ve) {
			return &statusError{status: se.status, err: se.err, fields: ve.Fields()}
		}
		return se
	case errors.As(err, &ve):
		return &statusError{status: http.StatusUnprocessableEntity, err: err, fields: ve.Fields()}
	case apierrors.IsNotFound(err):
		return &statusError{status: http.StatusNotFound, err: err}
	case apierrors.IsAlreadyExists(err), apierrors.IsConflict(err):
		return &statusError{status: http.StatusConflict, err: err}
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		return &statusError{status: http.StatusUnprocessableEntity, err: err}
	}
	for _, m := range domainErrors {
		if errors.Is(err, m.err) {
			return &statusError{status: m.status, err: err}
		}
	}
	return &statusError{status: http.StatusInternalServerError, err: err}
}

// domainErrors are the errors of the domain packages and their HTTP status.
var domainErrors = []struct {
	err    error
	status int
}{
	{egglibrary.ErrNotFound, http.StatusNotFound},
	{eggstore.ErrInvalid, http.StatusUnprocessableEntity},
	{eggstore.ErrInvalidURL, http.StatusUnprocessableEntity},
	{eggstore.ErrNoUpdateURL, http.StatusConflict},
	{eggstore.ErrDownload, http.StatusBadGateway},
	{eggstore.ErrInUse, http.StatusConflict},
	{eggstore.ErrNameTaken, http.StatusConflict},
	{files.ErrNotFound, http.StatusNotFound},
	{kube.ErrUserLimit, http.StatusTooManyRequests},
	{kube.ErrBusy, http.StatusServiceUnavailable},
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
	{selfupgrade.ErrRunning, http.StatusConflict},
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
	{settings.ErrNotConfigured, http.StatusUnprocessableEntity},
	{settings.ErrNotEnabled, http.StatusUnprocessableEntity},
	{settings.ErrNoCilium, http.StatusUnprocessableEntity},
	{users.ErrLastAdmin, http.StatusConflict},
	{users.ErrOwnAccount, http.StatusConflict},
	{users.ErrInvalidInvite, http.StatusNotFound},
	{users.ErrManaged, http.StatusConflict},
	{users.ErrNoPassword, http.StatusConflict},
}
