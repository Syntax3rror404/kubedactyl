package egglibrary

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"app/internal/testutil"
)

// archive builds a repository archive like codeload.github.com sends it.
func archive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		h := &tar.Header{Name: "eggs-HEAD/" + name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// testLibrary serves one repository (owner/eggs) with the Paper egg in both formats.
func testLibrary(t *testing.T) (*Library, *atomic.Int32) {
	t.Helper()
	files := map[string][]byte{
		"minecraft/paper/egg-paper.yaml":             testutil.Download(t, testutil.PaperPLCN),
		"minecraft/paper/pterodactyl-egg-paper.json": testutil.Download(t, testutil.PaperPTDL),
		".github/ISSUE_TEMPLATE/egg-request.yml":     []byte("name: Egg request\nbody: []\n"),
		"README.md":                                  []byte("# eggs"),
	}
	tarball := archive(t, files)
	var downloads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path, _ := strings.CutPrefix(r.URL.Path, "/raw/"); {
		case r.URL.Path == "/archive/owner/eggs":
			downloads.Add(1)
			_, _ = w.Write(tarball)
		case files[path] != nil:
			_, _ = w.Write(files[path])
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	l := New()
	l.Archive = func(owner, repo string) string { return srv.URL + "/archive/" + owner + "/" + repo }
	l.Raw = func(_, _, file string) string { return srv.URL + "/raw/" + file }
	return l, &downloads
}

func TestList(t *testing.T) {
	l, downloads := testLibrary(t)
	ctx := context.Background()
	repos := []string{"https://github.com/owner/eggs", "https://github.com/owner/missing"}
	got := l.List(ctx, repos, false)
	if len(got) != 2 || got[0].Error != "" || got[1].Error == "" {
		t.Fatalf("repositories = %+v", got)
	}
	// One egg: the Pterodactyl copy next to the Pelican file and the issue template are dropped.
	eggs := got[0].Eggs
	if len(eggs) != 1 || eggs[0].Path != "minecraft/paper/egg-paper.yaml" || eggs[0].Name != "Paper" ||
		!strings.HasPrefix(eggs[0].Format, "PLCN") || !strings.HasSuffix(eggs[0].URL, "/raw/"+eggs[0].Path) ||
		eggs[0].Tags == nil {
		t.Fatalf("eggs = %+v", eggs)
	}
	l.List(ctx, repos, false)
	if n := downloads.Load(); n != 1 {
		t.Errorf("cached repository downloaded %d times", n)
	}
	l.List(ctx, repos, true)
	if n := downloads.Load(); n != 2 {
		t.Errorf("refresh must download again (%d downloads)", n)
	}
}

func TestGet(t *testing.T) {
	l, _ := testLibrary(t)
	ctx := context.Background()
	repos := []string{"https://github.com/owner/eggs"}
	spec, e, err := l.Get(ctx, repos, repos[0], "minecraft/paper/egg-paper.yaml")
	if err != nil || spec.DisplayName != "Paper" || e.Name != "Paper" {
		t.Fatalf("get: %v %+v", err, e)
	}
	for _, c := range []struct{ repo, path string }{
		{repos[0], "minecraft/paper/pterodactyl-egg-paper.json"}, // dropped as duplicate
		{repos[0], "README.md"},
		{"https://github.com/other/eggs", "minecraft/paper/egg-paper.yaml"}, // not configured
	} {
		if _, _, err := l.Get(ctx, repos, c.repo, c.path); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s %s: want ErrNotFound, got %v", c.repo, c.path, err)
		}
	}
}
