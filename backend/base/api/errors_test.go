package api

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app/base/selfupgrade"
	"app/base/users"
	"app/base/validation"
)

func TestStatusOf(t *testing.T) {
	a := &API{}
	cases := []struct {
		err    error
		status int
		fields bool
	}{
		{BadRequest(errors.New("x")), http.StatusBadRequest, false},
		{validation.Errors{"domain": "invalid"}, http.StatusUnprocessableEntity, true},
		{users.ErrLastAdmin, http.StatusConflict, false},
		{selfupgrade.ErrRunning, http.StatusConflict, false},
		{errors.New("boom"), http.StatusInternalServerError, false},
	}
	for _, tc := range cases {
		se := a.statusOf(tc.err)
		if se.status != tc.status || (len(se.fields) > 0) != tc.fields {
			t.Errorf("%v: got %d fields=%v, want %d fields=%v", tc.err, se.status, se.fields, tc.status, tc.fields)
		}
	}
}

// Every error response goes through API.Fail, so all of them look the same and internal
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
		if strings.Contains(string(b), "ErrorResponse{") {
			t.Errorf("%s writes an error response itself: return an error and call a.Fail(w, r, err)", f)
		}
	}
}
