package files

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// localRoot returns a server root in a temporary folder (links resolved, so inside accepts it).
func localRoot(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestTreeScript: every path below the entries is listed, links are not followed.
func TestTreeScript(t *testing.T) {
	root := localRoot(t)
	mustMkdir(t, filepath.Join(root, "dir", "sub"))
	mustWrite(t, filepath.Join(root, "dir", "sub", "b.txt"), "b")
	mustWrite(t, filepath.Join(root, "dir", "a.txt"), "a")
	mustMkdir(t, filepath.Join(root, "other"))
	mustWrite(t, filepath.Join(root, "other", "secret.yml"), "x")
	if err := os.Symlink(filepath.Join(root, "other"), filepath.Join(root, "dir", "lnk")); err != nil {
		t.Fatal(err)
	}
	out, err := runScript(root, treeScript, filepath.Join(root, "dir"), "./sub", "./lnk")
	if err != nil {
		t.Fatal(err)
	}
	got := splitNul(out)
	slices.Sort(got)
	if want := []string{"./lnk", "./sub", "./sub/b.txt"}; !slices.Equal(got, want) {
		t.Errorf("tree = %q, want %q", got, want)
	}
	// A link as folder has nothing below it.
	if out, err := runScript(root, treeScript, filepath.Join(root, "dir", "lnk"), "."); err != nil || out != "" {
		t.Errorf("tree of a link = %q, %v", out, err)
	}
	if _, err := runScript(root, treeScript, filepath.Join(root, "dir"), "./missing"); exitCode(err) != 3 {
		t.Errorf("missing entry: %v", err)
	}
}

// TestCheckBelow: a path is checked below every root, where it is and where it will be.
func TestCheckBelow(t *testing.T) {
	deny := []string{"config/secret.yml", "*.key"}
	cases := []struct {
		paths, roots []string
		want         error
	}{
		{[]string{".", "./secret.yml"}, []string{"/world"}, nil},
		{[]string{".", "./secret.yml"}, []string{"/world", "/config"}, ErrDenied},
		{[]string{"./a/b.key"}, []string{"/"}, ErrDenied},
		{[]string{"./config/secret.yml"}, []string{"/"}, ErrDenied},
		{nil, []string{"/config"}, nil},
	}
	for _, c := range cases {
		if err := checkBelow(c.paths, c.roots, deny); err != c.want {
			t.Errorf("checkBelow(%q, %q) = %v, want %v", c.paths, c.roots, err, c.want)
		}
	}
}

// TestCompressScript: an archive is created once; an existing file of that name stays untouched
// (exit code 5) and no temporary file is left.
func TestCompressScript(t *testing.T) {
	root := localRoot(t)
	mustWrite(t, filepath.Join(root, "a.txt"), "a")
	if _, err := runScript(root, compressScript, root, "out.tar.gz", "./a.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "out.tar.gz")); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
	mustWrite(t, filepath.Join(root, "taken.tar.gz"), "keep")
	if _, err := runScript(root, compressScript, root, "taken.tar.gz", "./a.txt"); exitCode(err) != 5 {
		t.Errorf("existing archive: %v, want exit code 5", err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "taken.tar.gz")); string(got) != "keep" {
		t.Errorf("existing file overwritten: %q", got)
	}
	if left, _ := filepath.Glob(filepath.Join(root, ".*.partial")); len(left) > 0 {
		t.Errorf("temporary files left: %v", left)
	}
}

// TestSweep: leftovers of jobs older than jobLimit are removed, everything else stays. The local
// date runs two hours ahead; without GNU stat a shim reads the change time with BSD stat.
func TestSweep(t *testing.T) {
	if !strings.Contains(jobFuncs, fmt.Sprintf("-gt %d ]", int(jobLimit.Seconds()))) {
		t.Fatal("sweep must use jobLimit")
	}
	root := localRoot(t)
	leftovers := []string{".a.tar.gz.123.partial", ".x.jar.download", ".x.jar.download.log", ".extract-ABC"}
	kept := []string{"keep.txt", ".hidden", "a.partial"}
	for _, f := range append(leftovers, kept...) {
		mustWrite(t, filepath.Join(root, f), "x")
	}
	sweep := func(path string) {
		t.Helper()
		cmd := exec.Command("bash", "-c", jobFuncs+`sweep "$1"`, "sh", root)
		cmd.Env = append(os.Environ(), "PATH="+path)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sweep: %v %s", err, out)
		}
	}
	sweep(os.Getenv("PATH"))
	if left := present(root, leftovers); len(left) != len(leftovers) {
		t.Fatalf("fresh leftovers removed, left: %v", left)
	}
	sweep(shims(t) + ":" + os.Getenv("PATH"))
	if left := present(root, leftovers); len(left) > 0 {
		t.Errorf("old leftovers kept: %v", left)
	}
	if left := present(root, kept); len(left) != len(kept) {
		t.Errorf("other files removed, left: %v", left)
	}
}

// shims returns a folder with a date two hours ahead and, where stat has no -c, a GNU stat shim.
func shims(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	date := "#!/bin/sh\necho $(( $(/bin/date +%s) + 7200 ))\n"
	if err := os.WriteFile(filepath.Join(dir, "date"), []byte(date), 0o755); err != nil {
		t.Fatal(err)
	}
	if exec.Command("stat", "-c", "%Z", "/").Run() != nil {
		stat := "#!/bin/sh\nexec /usr/bin/stat -f %c \"$4\"\n"
		if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func present(root string, names []string) []string {
	var out []string
	for _, n := range names {
		if _, err := os.Lstat(filepath.Join(root, n)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func exitCode(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return 0
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}
