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

func TestRules(t *testing.T) {
	e := Errors{}
	e.Require("name", true)
	e.Require("owner", false)
	e.MaxLength("note", "äöü", 3) // characters, not bytes
	e.MaxLength("title", "four", 3)
	e.OneOf("signal", "", "start", "stop") // empty: not set
	e.OneOf("policy", "Sideways", "Local", "Cluster")
	want := Errors{
		"owner": "is required", "title": "must be at most 3 characters", "policy": "must be one of: Local, Cluster",
	}
	if len(e) != len(want) {
		t.Fatalf("errors %v, want %v", e, want)
	}
	for k, v := range want {
		if e[k] != v {
			t.Errorf("%s: %q, want %q", k, e[k], v)
		}
	}
}
