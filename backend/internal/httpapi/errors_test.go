package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app/api/v1alpha1"
	"app/base/api"
	"app/base/selfupgrade"
	"app/base/users"
	"app/base/validation"
	"app/internal/appsettings"
	"app/internal/egg"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/schedule"
	"app/internal/serverctl"
	"app/internal/testutil"
)

func TestStatusOf(t *testing.T) {
	fe := egg.FieldErrors{"name": "required"}
	cases := []struct {
		err    error
		status int
		fields bool
	}{
		{api.BadRequest(errors.New("x")), http.StatusBadRequest, false},
		{fe, http.StatusUnprocessableEntity, true},
		{validation.Errors{"domain": "invalid"}, http.StatusUnprocessableEntity, true},
		{users.ErrLastAdmin, http.StatusConflict, false},
		{fmt.Errorf("%w: x", eggstore.ErrDownload), http.StatusBadGateway, false},
		{schedule.ErrNotFound, http.StatusNotFound, false},
		{fmt.Errorf("wrapped: %w", files.ErrNotFound), http.StatusNotFound, false},
		{serverctl.ErrOffline, http.StatusConflict, false},
		{serverctl.ErrSuspended, http.StatusForbidden, false},
		{files.ErrBusy, http.StatusConflict, false},
		{files.ErrIntoItself, http.StatusUnprocessableEntity, false},
		{selfupgrade.ErrRunning, http.StatusConflict, false},
		{errors.New("boom"), http.StatusInternalServerError, false},
	}
	for _, tc := range cases {
		status, fields := statusOf(tc.err)
		if status != tc.status || (len(fields) > 0) != tc.fields {
			t.Errorf("%v: got %d fields=%v, want %d fields=%v", tc.err, status, fields, tc.status, tc.fields)
		}
	}
}

// statusOf answers a request with err (API.Fail, with the app's errors) and returns the status and the fields.
func statusOf(err error) (int, map[string]string) {
	a := &API{API: &api.API{Log: testutil.Logger()}}
	a.API.App = a
	rec := httptest.NewRecorder()
	a.Fail(rec, httptest.NewRequest(http.MethodGet, "/", nil), err)
	var body api.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body.Fields
}

// Every error response goes through API.fail, so all of them look the same and internal
// details are hidden from users in one place.
func TestErrorResponsesOnlyInFail(t *testing.T) {
	sources, _ := filepath.Glob("*.go")
	for _, f := range sources {
		if f == "errors.go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "api.ErrorResponse{") {
			t.Errorf("%s writes an error response itself: return an error and call a.Fail(w, r, err)", f)
		}
	}
}

// Domain errors keep their message when they are mapped to a status.
func TestDomainErrorMessages(t *testing.T) {
	_, err := appsettings.Choose("fast", "", []string{"longhorn"}, "storage class")
	if status, _ := statusOf(err); status != http.StatusUnprocessableEntity ||
		err.Error() != `storage class "fast" is not enabled in the panel settings` {
		t.Errorf("Choose: %d %q", status, err)
	}
	_, err = schedule.Validate([]v1alpha1.Schedule{{Name: "x", Cron: "not a cron"}})
	if status, _ := statusOf(err); status != http.StatusUnprocessableEntity ||
		strings.HasPrefix(err.Error(), "invalid schedule") {
		t.Errorf("schedule.Validate: %d %q", status, err)
	}
	_, err = files.PullName("https://example.com/", "")
	if status, _ := statusOf(err); status != http.StatusUnprocessableEntity {
		t.Errorf("PullName: %d %q", status, err)
	}
}
