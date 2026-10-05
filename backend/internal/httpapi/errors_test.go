package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app/api/v1alpha1"
	"app/internal/egg"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/schedule"
	"app/internal/selfupgrade"
	"app/internal/serverctl"
	"app/internal/settings"
	"app/internal/users"
	"app/internal/validation"
)

func TestStatusOf(t *testing.T) {
	fe := egg.FieldErrors{"name": "required"}
	cases := []struct {
		err    error
		status int
		fields bool
	}{
		{badRequest(errors.New("x")), http.StatusBadRequest, false},
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
		se := statusOf(tc.err)
		if se.status != tc.status || (len(se.fields) > 0) != tc.fields {
			t.Errorf("%v: got %d fields=%v, want %d fields=%v", tc.err, se.status, se.fields, tc.status, tc.fields)
		}
	}
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
		for _, bad := range []string{"ErrorResponse{", `gin.H{"error"`} {
			if strings.Contains(string(b), bad) {
				t.Errorf("%s writes an error response itself (%s): return an error and call a.fail(c, err)", f, bad)
			}
		}
	}
}

// Domain errors keep their message when they are mapped to a status.
func TestDomainErrorMessages(t *testing.T) {
	_, err := settings.Choose("fast", "", []string{"longhorn"}, "storage class")
	if se := statusOf(
		err,
	); se.status != http.StatusUnprocessableEntity ||
		err.Error() != `storage class "fast" is not enabled in the panel settings` {
		t.Errorf("Choose: %d %q", se.status, err)
	}
	_, err = schedule.Validate([]v1alpha1.Schedule{{Name: "x", Cron: "not a cron"}})
	if se := statusOf(
		err,
	); se.status != http.StatusUnprocessableEntity ||
		strings.HasPrefix(err.Error(), "invalid schedule") {
		t.Errorf("schedule.Validate: %d %q", se.status, err)
	}
	_, err = files.PullName("https://example.com/", "")
	if se := statusOf(err); se.status != http.StatusUnprocessableEntity {
		t.Errorf("PullName: %d %q", se.status, err)
	}
}
