package egglibrary

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// testLibrary serves two repositories of a self-hosted server with the Paper egg in both formats:
// owner/eggs in the Gitea layout and group/sub/eggs (a GitLab subgroup) in the GitLab layout.
// It returns the library, the server URL and the number of archive downloads.
func testLibrary(t *testing.T) (*Library, string, *atomic.Int32) {
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
		file, raw := strings.CutPrefix(r.URL.Path, "/owner/eggs/raw/HEAD/")
		if !raw {
			file, raw = strings.CutPrefix(r.URL.Path, "/group/sub/eggs/-/raw/HEAD/")
		}
		switch {
		case r.URL.Path == "/owner/eggs/archive/HEAD.tar.gz",
			r.URL.Path == "/group/sub/eggs/-/archive/HEAD/eggs-HEAD.tar.gz":
			downloads.Add(1)
			_, _ = w.Write(tarball)
		case raw && files[file] != nil:
			_, _ = w.Write(files[file])
		case r.URL.Path == "/html/page/get/HEAD.tar.gz":
			_, _ = w.Write([]byte("<html>sign in</html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return New(), srv.URL, &downloads
}

func TestHostURLs(t *testing.T) {
	for _, c := range []struct{ repo, archive, raw string }{
		{"https://github.com/o/r", "https://codeload.github.com/o/r/tar.gz/HEAD",
			"https://raw.githubusercontent.com/o/r/HEAD/a/egg.json"},
		{"https://gitlab.com/g/s/r", "https://gitlab.com/g/s/r/-/archive/HEAD/r-HEAD.tar.gz",
			"https://gitlab.com/g/s/r/-/raw/HEAD/a/egg.json"},
		{"https://codeberg.org/o/r", "https://codeberg.org/o/r/archive/HEAD.tar.gz",
			"https://codeberg.org/o/r/raw/HEAD/a/egg.json"},
		{"https://bitbucket.org/o/r", "https://bitbucket.org/o/r/get/HEAD.tar.gz",
			"https://bitbucket.org/o/r/raw/HEAD/a/egg.json"},
	} {
		u, _ := url.Parse(c.repo)
		h := hostsOf(u)
		if len(h) != 1 || h[0].archive(u) != c.archive || h[0].raw(u, "a/egg.json") != c.raw {
			t.Errorf("%s: %d layouts, %s, %s", c.repo, len(h), h[0].archive(u), h[0].raw(u, "a/egg.json"))
		}
	}
}

func TestList(t *testing.T) {
	l, base, downloads := testLibrary(t)
	ctx := context.Background()
	repos := []string{base + "/owner/eggs", base + "/owner/missing", base + "/html/page"}
	got := l.List(ctx, repos, false)
	if len(got) != 3 || got[0].Error != "" || got[1].Error != errUnknownHost.Error() || got[2].Error == "" {
		t.Fatalf("repositories = %+v", got)
	}
	// One egg: the Pterodactyl copy next to the Pelican file and the issue template are dropped.
	eggs := got[0].Eggs
	if len(eggs) != 1 || eggs[0].Path != "minecraft/paper/egg-paper.yaml" || eggs[0].Name != "Paper" ||
		!strings.HasPrefix(eggs[0].Format, "PLCN") || eggs[0].URL != repos[0]+"/raw/HEAD/"+eggs[0].Path ||
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
	l, base, _ := testLibrary(t)
	ctx := context.Background()
	repos := []string{base + "/owner/eggs", base + "/group/sub/eggs"}
	spec, e, err := l.Get(ctx, repos, repos[0], "minecraft/paper/egg-paper.yaml")
	if err != nil || spec.DisplayName != "Paper" || e.Name != "Paper" {
		t.Fatalf("get: %v %+v", err, e)
	}
	// The GitLab layout of the subgroup is found by trying.
	if _, e, err := l.Get(ctx, repos, repos[1], "minecraft/paper/egg-paper.yaml"); err != nil ||
		e.URL != repos[1]+"/-/raw/HEAD/minecraft/paper/egg-paper.yaml" {
		t.Fatalf("get from the subgroup: %v %+v", err, e)
	}
	for _, c := range []struct{ repo, path string }{
		{repos[0], "minecraft/paper/pterodactyl-egg-paper.json"}, // dropped as duplicate
		{repos[0], "README.md"},
		{base + "/other/eggs", "minecraft/paper/egg-paper.yaml"}, // not configured
	} {
		if _, _, err := l.Get(ctx, repos, c.repo, c.path); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s %s: want ErrNotFound, got %v", c.repo, c.path, err)
		}
	}
}

func TestGetRepository(t *testing.T) {
	l, base, downloads := testLibrary(t)
	ctx := context.Background()
	r := l.GetRepository(ctx, base+"/owner/eggs")
	if r.Error != "" || len(r.Eggs) != 1 {
		t.Fatalf("repository = %+v", r)
	}
	if r := l.GetRepository(ctx, base+"/owner/missing"); r.Error == "" {
		t.Errorf("missing repository: %+v", r)
	}
	// Checked before it was saved: the library lists it without downloading it again.
	l.List(ctx, []string{base + "/owner/eggs"}, false)
	if n := downloads.Load(); n != 1 {
		t.Errorf("checked repository downloaded %d times", n)
	}
}
