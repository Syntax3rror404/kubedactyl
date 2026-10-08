package httpapi

import (
	"testing"
)

// The pool check itself is settings.Pool.Contains; without a pool only the format is checked.
func TestCheckPoolIPFormat(t *testing.T) {
	a := &API{}
	for ips, ok := range map[string]bool{
		"":                          true,
		"192.168.1.70":              true,
		"2001:db8::5":               true,
		"192.168.1.70, 2001:db8::5": true,
		"192.168.1.70,192.168.1.71": false,
		"2001:db8::5,2001:db8::6":   false,
		"192.168.1.70,":             false,
		"not an ip":                 false,
	} {
		if err := a.checkPoolIP(t.Context(), "", ips); (err == nil) != ok {
			t.Errorf("%q: %v", ips, err)
		}
	}
}
