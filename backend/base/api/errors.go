package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"app/base/kube"
	"app/base/selfupgrade"
	"app/base/users"
	"app/base/validation"
)

// Every failed request is answered the same way: a.Fail(c, err).
//
// Handlers and helpers choose the status by wrapping the error (BadRequest(err), Forbidden(err), …).
// Errors without a status are mapped by statusOf: validation errors (validation.Error, 422 with the
// messages per field), Kubernetes errors, the errors of the base packages (domainErrors) and those of the app
// (App.StatusOf).
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

func BadRequest(err error) error      { return httpError(http.StatusBadRequest, err) }
func Unauthorized(err error) error    { return httpError(http.StatusUnauthorized, err) }
func Forbidden(err error) error       { return httpError(http.StatusForbidden, err) }
func NotFound(err error) error        { return httpError(http.StatusNotFound, err) }
func Conflict(err error) error        { return httpError(http.StatusConflict, err) }
func TooManyRequests(err error) error { return httpError(http.StatusTooManyRequests, err) }

// fieldError is an error of one form field; the frontend shows it next to the field.
func FieldError(status int, field string, err error) error {
	return &statusError{status: status, err: err, fields: map[string]string{field: err.Error()}}
}

// fail answers a failed request and aborts the handler chain.
func (a *API) Fail(c *gin.Context, err error) {
	se := a.statusOf(err)
	msg := se.err.Error()
	if se.status == http.StatusInternalServerError {
		a.Log.Error("request failed", "path", c.FullPath(), "err", err)
		// Details (object names, cluster errors) only for administrators.
		if p := PrincipalOf(c); p == nil || !p.Admin() {
			msg = "internal error: the administrator can find details in the panel log"
		}
	}
	c.AbortWithStatusJSON(se.status, ErrorResponse{Error: msg, Fields: se.fields})
}

// bindJSON reads the request body into req; a body that does not fit is answered with 400.
func (a *API) BindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		a.Fail(c, BadRequest(err))
		return false
	}
	return true
}

// HTTPStatus is the status a.Fail answers err with.
func (a *API) HTTPStatus(err error) int { return a.statusOf(err).status }

// statusOf finds the HTTP status of an error.
func (a *API) statusOf(err error) *statusError {
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
	if a.App != nil {
		if status := a.App.StatusOf(err); status != 0 {
			return &statusError{status: status, err: err}
		}
	}
	return &statusError{status: http.StatusInternalServerError, err: err}
}

// domainErrors are the errors of the base packages and their HTTP status.
var domainErrors = []struct {
	err    error
	status int
}{
	{kube.ErrUserLimit, http.StatusTooManyRequests},
	{kube.ErrBusy, http.StatusServiceUnavailable},
	{selfupgrade.ErrRunning, http.StatusConflict},
	{users.ErrLastAdmin, http.StatusConflict},
	{users.ErrOwnAccount, http.StatusConflict},
	{users.ErrInvalidInvite, http.StatusNotFound},
	{users.ErrManaged, http.StatusConflict},
	{users.ErrNoPassword, http.StatusConflict},
	{users.ErrTokenLimit, http.StatusConflict},
}
