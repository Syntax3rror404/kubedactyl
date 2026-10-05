package egg

import (
	"reflect"
	"testing"
)

func TestSplitRules(t *testing.T) {
	got := SplitRules(`required|regex:/^(release|snapshot)$/|max:20`)
	want := []string{"required", "regex:/^(release|snapshot)$/", "max:20"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		value, rules string
		ok           bool
	}{
		{"", "required|string", false},
		{"", "nullable|string", true},
		{"server.jar", `required|regex:/^([\w\d._-]+)(\.jar)$/`, true},
		{"server.zip", `required|regex:/^([\w\d._-]+)(\.jar)$/`, false},
		{"snapshot", `required|regex:/^(release|snapshot)$/`, true},
		{"latest", "required|string|max:20", true},
		{"this value is way too long", "required|string|max:20", false},
		{"25", "required|integer|between:1,100", true},
		{"250", "required|integer|between:1,100", false},
		{"1", "required|boolean", true},
		{"yes", "required|boolean", false},
		{"vanilla", "required|in:vanilla,forge", true},
		{"paper", "required|in:vanilla,forge", false},
		{"12345", "digits_between:3,5", true},
		{"my_world-1", "alpha_dash", true},
		{"my world", "alpha_dash", false},
		{"https://example.com/a.jar", "nullable|url", true},
		{"ABC", `regex:/^abc$/i`, true},
		{"1.5", "numeric", true},
		{"abc", "numeric", false},
		{"10", "numeric|gt:9", true},
		{"9", "numeric|gt:9", false},
		{"abcd", "size:4", true},
		{"abc", "size:4", false},
		{"ab", "min:3", false},
		{"forge", "not_in:vanilla,forge", false},
		{"world.zip", "ends_with:.zip,.tar", true},
		{"world.rar", "ends_with:.zip,.tar", false},
		{"admin", `not_regex:/^admin$/`, false},
		{"abc123", "alpha_num", true},
		{"abc-123", "alpha_num", false},
		{"anything", "string|unknown_rule:5", true},
	}
	for _, c := range cases {
		if err := Validate(c.value, c.rules); (err == nil) != c.ok {
			t.Errorf("Validate(%q, %q) = %v, want ok=%v", c.value, c.rules, err, c.ok)
		}
	}
}
