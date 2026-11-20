package httpserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/swaggo/swag"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"app/api/v1alpha1"
	"app/base/api"
	"app/base/bootstrap"
	"app/base/settings"
)

// The app registers its generated documentation; the tests serve a stub.
func init() {
	swag.Register(swag.Name, &swag.Spec{SwaggerTemplate: `{"swagger": "2.0"}`})
}

// routes stands in for the app's API: one route that passes on another server's answer.
type routes struct{}

func (routes) Register(m *api.Mux) {
	m.Handle("GET /proxy", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Del("Cache-Control")
		w.WriteHeader(http.StatusOK)
	})
	m.Handle("GET /panic", func(http.ResponseWriter, *http.Request) { panic("boom") })
	// A route with a credential in its path, like a forwarded port of another app.
	m.Handle("GET /forward/{token}/{path...}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	// Answers whether the request context ended (the client went away).
	contextEnded := func(w http.ResponseWriter, r *http.Request) {
		api.JSON(w, http.StatusOK, r.Context().Err() != nil)
	}
	m.Handle("GET /context", contextEnded)
	// Read the whole body, like a handler that decodes JSON or an upload.
	readBody := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		}
	}
	m.Handle("POST /echo/{name}", readBody)
	m.Handle("GET /echo/{name}", readBody)
	m.Handle("POST /{server}/files/upload", readBody)
	m.Handle("POST /context", contextEnded)
}

// newRouter builds the router with fresh settings and the frontend (nil: development mode).
func newRouter(t *testing.T, frontend fs.FS) (http.Handler, *settings.Store) {
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
	if code := get("/api/openapi.json"); code != http.StatusOK {
		t.Fatalf("API document by default: %d", code)
	}
	// Saved through the store, like the settings page does: applies at once.
	if _, err := store.Update(t.Context(), v1alpha1.PanelSettingsSpec{DisableAPIDocs: true}); err != nil {
		t.Fatal(err)
	}
	if code := get("/api/openapi.json"); code != http.StatusNotFound {
		t.Errorf("API document with the API docs turned off: %d", code)
	}
	if code := get("/api/health"); code != http.StatusOK {
		t.Errorf("the API keeps working: %d", code)
	}
}

// TestAPIAnswersAreNotCached: answers of the API may hold tokens and file contents; a handler that passes on
// another server's answer keeps that server's caching.
func TestAPIAnswersAreNotCached(t *testing.T) {
	r, _ := newRouter(t, nil)
	for path, want := range map[string]string{"/api/health": "no-store", "/api/proxy": ""} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if got := w.Header().Get("Cache-Control"); got != want {
			t.Errorf("%s: Cache-Control %q, want %q", path, got, want)
		}
	}
}

// TestWebUIIsCachedAndCompressed: hashed assets are cached for a year, text files are sent gzip compressed to
// browsers that accept it, unknown paths (also with a trailing slash) and folders get index.html, missing assets a
// 404.
func TestWebUIIsCachedAndCompressed(t *testing.T) {
	script := strings.Repeat("console.log('panel');", 100)
	// Below a folder like the embedded build (fs.Sub), which rejects paths with a trailing slash as invalid.
	build, err := fs.Sub(fstest.MapFS{
		"dist/index.html":            {Data: []byte("<!doctype html><title>Panel</title>")},
		"dist/assets/index-abc.js":   {Data: []byte(script)},
		"dist/assets/font-abc.woff2": {Data: []byte("binary font")},
	}, "dist")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := newRouter(t, build)
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
	for _, path := range []string{"/", "/some/page", "/some/page/", "/assets/", "/a//b", "/api-docs"} {
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

// TestUnknownAPIPaths: unknown API paths are a JSON 404, never the web UI or a redirect.
func TestUnknownAPIPaths(t *testing.T) {
	r, _ := newRouter(t, nil)
	for _, path := range []string{"/api", "/api/", "/api/nope", "/api/health/extra", "/api/x%2F..%2Fhealth"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), `"error"`) {
			t.Errorf("%s: %d %q", path, w.Code, w.Body.String())
		}
	}
}

// TestBodyLimitByRoute: only the upload route takes large bodies; a path that merely ends like it (an encoded
// slash in a path value) is limited, too.
func TestBodyLimitByRoute(t *testing.T) {
	r, _ := newRouter(t, nil)
	send := func(target string) *httptest.ResponseRecorder {
		method, path, _ := strings.Cut(target, " ")
		body := strings.NewReader(strings.Repeat("x", maxRequestBody+1))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, body))
		return w
	}
	// Also a read with a body is limited.
	for target, limited := range map[string]bool{
		"POST /api/echo/x": true, "GET /api/echo/x": true, "POST /api/echo/x%2Ffiles%2Fupload": true,
		"POST /api/x/files/upload": false,
	} {
		w := send(target)
		if strings.Contains(w.Body.String(), "too large") != limited || w.Code == http.StatusNotFound {
			t.Errorf("%s: %d %q, limited %v", target, w.Code, w.Body.String(), limited)
		}
	}
}

// TestTokensAreNotLogged: a credential in the path of a route ({token}) never reaches the log.
func TestTokensAreNotLogged(t *testing.T) {
	var log bytes.Buffer
	r, err := NewRouter(Config{API: routes{}, Log: slog.New(slog.NewTextHandler(&log, nil))})
	if err != nil {
		t.Fatal(err)
	}
	// Also the start page of a forward ("/" behind the token) reaches the route.
	for _, path := range []string{"/api/forward/secret-token/x", "/api/forward/secret-token/"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
	if !strings.Contains(log.String(), "/api/forward/{token}/{path...}") || strings.Contains(log.String(), "secret") {
		t.Errorf("log: %s", log.String())
	}
}

// TestPanicsAreAnswered: a handler that panics gets a 500 instead of a closed connection.
func TestPanicsAreAnswered(t *testing.T) {
	r, _ := newRouter(t, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/panic", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic: %d", w.Code)
	}
}

// TestChangesFinish: a request that changes something keeps its context when the client goes away (a change is
// never left half done); a read stops with the client.
func TestChangesFinish(t *testing.T) {
	r, _ := newRouter(t, nil)
	gone, cancel := context.WithCancel(t.Context())
	cancel()
	for method, want := range map[string]string{http.MethodPost: "false", http.MethodGet: "true"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequestWithContext(gone, method, "/api/context", nil))
		if got := strings.TrimSpace(w.Body.String()); got != want {
			t.Errorf("%s: context ended %s, want %s", method, got, want)
		}
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
