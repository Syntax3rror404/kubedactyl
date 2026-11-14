package api

import "testing"

func TestLocalPath(t *testing.T) {
	for next, want := range map[string]string{
		"/servers/a?tab=files":                 "/servers/a?tab=files",
		"":                                     "/",
		"//evil.example.com":                   "/",
		"/\\evil.example.com":                  "/",
		"https://evil.example":                 "/",
		"/api/servers/a/files/download?file=x": "/",
		"servers":                              "/",
		"/a\nb":                                "/",
	} {
		if got := localPath(next); got != want {
			t.Errorf("localPath(%q) = %q, want %q", next, got, want)
		}
	}
}
