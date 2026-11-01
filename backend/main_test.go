package main

import "testing"

func TestLocalURL(t *testing.T) {
	cases := []struct{ addr, want string }{
		{":8080", "http://localhost:8080"},
		{"127.0.0.1:18090", "http://localhost:18090"},
		{"[::1]:8080", "http://localhost:8080"},
		{"no-port", "http://localhost"},
	}
	for _, c := range cases {
		if got := localURL(c.addr); got != c.want {
			t.Errorf("localURL(%q) = %q, want %q", c.addr, got, c.want)
		}
	}
}
