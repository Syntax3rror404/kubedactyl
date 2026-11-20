package main

import (
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/swaggo/swag"

	"app/base/api"
	"app/base/httpserver"
	"app/base/settings"
	"app/internal/httpapi"
	"app/internal/testutil"
)

// apiDoc is the generated API documentation (docs, registered by main.go's import).
type apiDoc struct {
	Info struct {
		Title string `json:"title"`
	} `json:"info"`
	BasePath    string                                `json:"basePath"`
	Paths       map[string]map[string]json.RawMessage `json:"paths"`
	Definitions map[string]struct {
		Properties map[string]json.RawMessage `json:"properties"`
	} `json:"definitions"`
}

// readDoc guards against swag silently serving the raw template: when the template cannot be executed (e.g.
// "{{...}}" in a description) ReadDoc returns it unrendered.
func readDoc(t *testing.T) apiDoc {
	t.Helper()
	doc, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}
	var spec apiDoc
	if err := json.Unmarshal([]byte(doc), &spec); err != nil {
		t.Fatalf("swagger doc is not valid JSON (template not rendered?): %v", err)
	}
	return spec
}

func TestSwaggerDocRenders(t *testing.T) {
	spec := readDoc(t)
	if spec.Info.Title != "Kubedactyl API" || spec.BasePath != "/api" {
		t.Errorf("info not rendered: title=%q basePath=%q", spec.Info.Title, spec.BasePath)
	}
	if len(spec.Paths) < 20 {
		t.Errorf("only %d paths documented", len(spec.Paths))
	}
	for _, name := range []string{"v1alpha1.GameServer", "v1alpha1.Egg", "httpapi.CreateServerRequest"} {
		if len(spec.Definitions[name].Properties) == 0 {
			t.Errorf("schema %s has no properties", name)
		}
	}
}

// TestDocumentedRoutesExist guards the @Router annotations: every path in the API
// documentation must be a registered route (a renamed handler once renamed its path, too).
func TestDocumentedRoutesExist(t *testing.T) {
	c := testutil.Builder(t).Build()
	store := &settings.Store{Client: c, Reader: c}
	mux := httpserver.NewAPI(httpserver.Config{
		API: &httpapi.API{API: &api.API{Settings: store}}, Settings: store, Log: testutil.Logger(),
	})
	spec := readDoc(t)
	param := regexp.MustCompile(`\{\w+\}`)
	for path, ops := range spec.Paths {
		for method := range ops {
			want := strings.ToUpper(method) + " " + spec.BasePath + path
			req := httptest.NewRequest(strings.ToUpper(method), spec.BasePath+param.ReplaceAllString(path, "x"), nil)
			if _, pattern := mux.Handler(req); pattern != want {
				t.Errorf("documented route %s is not registered (matched %q)", want, pattern)
			}
		}
	}
}
