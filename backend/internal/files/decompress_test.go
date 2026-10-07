package files

import (
	"archive/tar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"app/internal/gameserver"
)

// writeTar creates an archive; an entry with a "->" in its body is a symbolic link.
func writeTar(t *testing.T, file string, entries [][2]string) {
	t.Helper()
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	w := tar.NewWriter(f)
	for _, e := range entries {
		h := &tar.Header{Name: e[0], Mode: 0o644, Size: int64(len(e[1])), Typeflag: tar.TypeReg}
		if target, ok := strings.CutPrefix(e[1], "->"); ok {
			h = &tar.Header{Name: e[0], Mode: 0o777, Typeflag: tar.TypeSymlink, Linkname: target}
		}
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := w.Write([]byte(e[1])); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestDecompressScript runs the extraction with the local bash and tar (the files pod's BusyBox sh knows
// "set -o pipefail", the dash of Debian based systems does not): entries land in the archive's folder
// only, existing folders are merged, and a link in the way is replaced instead of written through.
func TestDecompressScript(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
	for _, d := range []string{filepath.Join(root, "sub"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "sub", "keep.txt"), "kept")
	mustWrite(t, filepath.Join(outside, "target.txt"), "outside")
	if err := os.Symlink(filepath.Join(outside, "target.txt"), filepath.Join(root, "a.txt")); err != nil {
		t.Fatal(err)
	}
	writeTar(t, filepath.Join(root, "ok.tar"), [][2]string{
		{"a.txt", "new a"}, {"sub/b.txt", "b"}, {"lnk", "->" + outside},
	})
	extract := func(archive string) error {
		script := strings.ReplaceAll(gameserver.PathScript, gameserver.ServerRoot, root) + decompressScript
		return exec.Command("bash", "-c", script, "sh", filepath.Join(root, archive)).Run()
	}
	if err := extract("ok.tar"); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]string{
		"root/a.txt": "new a", "root/sub/b.txt": "b", "root/sub/keep.txt": "kept", "outside/target.txt": "outside",
	} {
		if got, err := os.ReadFile(filepath.Join(base, file)); err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", file, got, err, want)
		}
	}
	// "../" entries and writes through a link of the same archive must not leave the folder.
	writeTar(t, filepath.Join(root, "evil.tar"), [][2]string{
		{"../escape.txt", "x"}, {"out", "->" + outside}, {"out/through.txt", "x"},
	})
	_ = extract("evil.tar")
	for _, file := range []string{"escape.txt", "outside/through.txt"} {
		if _, err := os.Stat(filepath.Join(base, file)); err == nil {
			t.Errorf("%s was written outside the folder", file)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(root, ".extract-*")); len(left) > 0 {
		t.Errorf("temporary folders left: %v", left)
	}
}

func mustWrite(t *testing.T, file, content string) {
	t.Helper()
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
