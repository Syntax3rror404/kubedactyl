package httpserver

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"app/api/v1alpha1"
	"app/base/bootstrap"
	"app/base/settings"
)

// The app registers its generated documentation; the tests serve a stub.
func init() {
	swag.Register(swag.Name, &swag.Spec{SwaggerTemplate: `{"swagger": "2.0"}`})
}

// routes stands in for the app's API: one route that passes on another server's answer.
type routes struct{}

func (routes) Register(r *gin.RouterGroup) {
	r.GET("/proxy", func(c *gin.Context) {
		c.Writer.Header().Del("Cache-Control")
		c.Status(http.StatusOK)
	})
}

// newRouter builds the router with fresh settings and the frontend (nil: development mode).
func newRouter(t *testing.T, frontend fstest.MapFS) (*gin.Engine, *settings.Store) {
	t.Helper()
	scheme, err := bootstrap.Scheme()
	if err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := &settings.Store{Client: c, Reader: c, Namespace: "panel"}
	cfg := Config{API: routes{}, Settings: store, Log: slog.New(slog.DiscardHandler)}
	if frontend != nil { // a nil map would be a frontend without files
		cfg.Frontend = frontend
	}
	r, err := NewRouter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return r, store
}

func TestAPIDocsCanBeTurnedOff(t *testing.T) {
	r, store := newRouter(t, nil)
	get := func(path string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w.Code
	}
	if code := get("/swagger/doc.json"); code != http.StatusOK {
		t.Fatalf("API docs by default: %d", code)
	}
	// Saved through the store, like the settings page does: applies at once.
	if _, err := store.Update(t.Context(), v1alpha1.PanelSettingsSpec{DisableAPIDocs: true}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json", "/swagger/"} {
		if code := get(path); code != http.StatusNotFound {
			t.Errorf("%s with the API docs turned off: %d", path, code)
		}
	}
	if code := get("/api/health"); code != http.StatusOK {
		t.Errorf("the API keeps working: %d", code)
	}
}

// TestAPIAnswersAreNotCached: answers of the API may hold tokens and file contents; a handler that passes on
// another server's answer keeps that server's caching.
func TestAPIAnswersAreNotCached(t *testing.T) {
	r, _ := newRouter(t, nil)
	for path, want := range map[string]string{"/api/health": "no-store", "/api/proxy": "", "/swagger/doc.json": ""} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if got := w.Header().Get("Cache-Control"); got != want {
			t.Errorf("%s: Cache-Control %q, want %q", path, got, want)
		}
	}
}

// TestWebUIIsCachedAndCompressed: hashed assets are cached for a year, text files are sent gzip compressed to
// browsers that accept it, unknown paths and folders get index.html, missing assets a 404.
func TestWebUIIsCachedAndCompressed(t *testing.T) {
	script := strings.Repeat("console.log('panel');", 100)
	r, _ := newRouter(t, fstest.MapFS{
		"index.html":            {Data: []byte("<!doctype html><title>Panel</title>")},
		"assets/index-abc.js":   {Data: []byte(script)},
		"assets/font-abc.woff2": {Data: []byte("binary font")},
	})
	get := func(path, encoding string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept-Encoding", encoding)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w := get("/assets/index-abc.js", "gzip, deflate, br")
	if w.Header().Get("Content-Encoding") != "gzip" || w.Header().Get("Vary") != "Accept-Encoding" ||
		w.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		!strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("compressed asset headers: %v", w.Header())
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	if body, _ := io.ReadAll(zr); string(body) != script {
		t.Error("compressed asset does not unpack to the file")
	}
	if w := get("/assets/index-abc.js", ""); w.Header().Get("Content-Encoding") != "" || w.Body.String() != script {
		t.Errorf("asset without gzip: %v", w.Header())
	}
	if w := get("/assets/font-abc.woff2", "gzip"); w.Header().Get("Content-Encoding") != "" {
		t.Errorf("fonts are compressed already: %v", w.Header())
	}
	for _, path := range []string{"/", "/some/page", "/assets/"} {
		w := get(path, "")
		if !strings.Contains(w.Body.String(), "<title>Panel") || w.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: %d %v %q", path, w.Code, w.Header(), w.Body.String())
		}
	}
	// A tab of an older build reloads on the 404 instead of reading index.html as a script.
	w = get("/assets/gone-abc.js", "")
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "<title>") {
		t.Errorf("missing asset: %d %q", w.Code, w.Body.String())
	}
	// Files without a hash in their name are revalidated: unchanged, the answer is 304 without a body.
	etag := get("/", "gzip").Header().Get("ETag")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if etag == "" || w.Code != http.StatusNotModified || w.Body.Len() != 0 {
		t.Errorf("revalidation with %q: %d %q", etag, w.Code, w.Body.String())
	}
}

func TestLocalURL(t *testing.T) {
	cases := []struct{ addr, want string }{
		{":8080", "http://localhost:8080"},
		{"127.0.0.1:18090", "http://localhost:18090"},
		{"[::1]:8080", "http://localhost:8080"},
		{"no-port", "http://localhost"},
	}
	for _, c := range cases {
		if got := LocalURL(c.addr); got != c.want {
			t.Errorf("LocalURL(%q) = %q, want %q", c.addr, got, c.want)
		}
	}
}
