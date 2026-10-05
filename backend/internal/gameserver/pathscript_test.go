package gameserver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runPathScript runs script after PathScript with root as the server root (local sh and realpath).
func runPathScript(t *testing.T, root, script string, args ...string) (string, int) {
	t.Helper()
	full := strings.ReplaceAll(PathScript, ServerRoot, root) + script
	out, err := exec.Command("sh", append([]string{"-c", full, "sh"}, args...)...).Output()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(out), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

// TestPathScript: links are resolved, and inside refuses every path that leads out of the root.
func TestPathScript(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
	for _, d := range []string{filepath.Join(root, "dir"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for link, target := range map[string]string{
		"out": outside, "in": "dir", "file": filepath.Join(outside, "x"), "missing": "dir/new",
		"up": "nope/../../outside/y", "loop": "loop", "rel": "missing.txt",
	} {
		if err := os.Symlink(target, filepath.Join(root, link)); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		script, path, want string
	}{
		{"real", "/dir/new/file", root + "/dir/new/file"},
		{"real", "/in/a", root + "/dir/a"},
		{"real", "/out/a", outside + "/a"},
		{"real", "/file", outside + "/x"},
		{"real", "/missing", root + "/dir/new"},
		{"real", "/up", ""},
		{"real", "/loop", ""},
		{"real", "/rel", root + "/missing.txt"},
		{"entry", "/out", root + "/out"},
		{"entry", "/in/a", root + "/dir/a"},
	} {
		got, code := runPathScript(t, root, c.script+` "$1"`, root+c.path)
		if (code != 0 && c.want != "") || strings.TrimSpace(got) != c.want {
			t.Errorf("%s %s = %q (exit %d), want %q", c.script, c.path, got, code, c.want)
		}
	}
	for path, want := range map[string]int{
		"/dir/new": 0, "/in/x": 0, "/out/x": 3, "/file": 3, "/../outside": 3, "/missing": 0, "/up": 3,
		"/loop": 3,
	} {
		if _, code := runPathScript(t, root, `inside "$(real "$1")"`, root+path); code != want {
			t.Errorf("inside %s: exit %d, want %d", path, code, want)
		}
	}
}
