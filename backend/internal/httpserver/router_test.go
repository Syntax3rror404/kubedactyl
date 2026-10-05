package httpserver

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/swaggo/swag"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/httpapi"
	"app/internal/settings"
	"app/internal/testutil"
)

func TestAPIDocsCanBeTurnedOff(t *testing.T) {
	longhorn := &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "longhorn"}}
	c := testutil.Builder(t).WithObjects(longhorn).Build()
	store := &settings.Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	r, err := NewRouter(Config{API: &httpapi.API{Settings: store}, Log: testutil.Logger()})
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w.Code
	}
	if code := get("/swagger/doc.json"); code != http.StatusOK {
		t.Fatalf("API docs by default: %d", code)
	}
	// Saved through the store, like the settings page does: applies at once.
	if _, err := store.Update(t.Context(), v1alpha1.PanelSettingsSpec{
		StorageClasses: []string{"longhorn"}, DisableAPIDocs: true,
	}); err != nil {
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

// TestAPIAnswersAreNotCached: answers of the API may hold tokens and file contents.
func TestAPIAnswersAreNotCached(t *testing.T) {
	c := testutil.Builder(t).Build()
	store := &settings.Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	r, err := NewRouter(Config{API: &httpapi.API{Settings: store}, Log: testutil.Logger()})
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{"/api/health": "no-store", "/swagger/doc.json": ""} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if got := w.Header().Get("Cache-Control"); got != want {
			t.Errorf("%s: Cache-Control %q, want %q", path, got, want)
		}
	}
}

// TestDocumentedRoutesExist guards the @Router annotations: every path in the API
// documentation must be a registered route (a renamed handler once renamed its path, too).
func TestDocumentedRoutesExist(t *testing.T) {
	c := testutil.Builder(t).Build()
	r, err := NewRouter(
		Config{API: &httpapi.API{Settings: &settings.Store{Client: c, Reader: c}}, Log: testutil.Logger()},
	)
	if err != nil {
		t.Fatal(err)
	}
	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	doc, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		BasePath string                                `json:"basePath"`
		Paths    map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal([]byte(doc), &spec); err != nil {
		t.Fatal(err)
	}
	param := regexp.MustCompile(`\{(\w+)\}`)
	for path, ops := range spec.Paths {
		for method := range ops {
			key := strings.ToUpper(method) + " " + spec.BasePath + param.ReplaceAllString(path, ":$1")
			if !registered[key] {
				t.Errorf("documented route %s is not registered", key)
			}
		}
	}
}

// TestWebUIIsCachedAndCompressed: hashed assets are cached for a year, text files are sent gzip compressed to
// browsers that accept it, unknown paths and folders get index.html.
func TestWebUIIsCachedAndCompressed(t *testing.T) {
	script := strings.Repeat("console.log('kubedactyl');", 100)
	frontend := fstest.MapFS{
		"index.html":            {Data: []byte("<!doctype html><title>Kubedactyl</title>")},
		"assets/index-abc.js":   {Data: []byte(script)},
		"assets/font-abc.woff2": {Data: []byte("binary font")},
	}
	c := testutil.Builder(t).Build()
	store := &settings.Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	r, err := NewRouter(Config{API: &httpapi.API{Settings: store}, Frontend: frontend, Log: testutil.Logger()})
	if err != nil {
		t.Fatal(err)
	}
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
	for _, path := range []string{"/", "/servers/x", "/assets/"} {
		w := get(path, "")
		if !strings.Contains(w.Body.String(), "<title>Kubedactyl") || w.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: %d %v %q", path, w.Code, w.Header(), w.Body.String())
		}
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
