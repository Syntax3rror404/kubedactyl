package validation

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	if (Errors{}).OrNil() != nil {
		t.Error("empty errors must be nil")
	}
	e := Errors{"b": "second", "a": "first"}
	if got := e.Error(); got != "a: first (and 1 more)" {
		t.Errorf("Error() = %q", got)
	}
	var ve Error
	if !errors.As(error(Field("name", errors.New("is required"))), &ve) || ve.Fields()["name"] != "is required" {
		t.Error("Field must be a validation.Error with the message")
	}
}
