package httpserver

import (
	"encoding/json"
	"testing"

	"github.com/swaggo/swag"

	_ "app/docs"
)

// TestSwaggerDocRenders guards against swag silently serving the raw template: when the
// template cannot be executed (e.g. "{{...}}" in a description) ReadDoc returns it unrendered.
func TestSwaggerDocRenders(t *testing.T) {
	doc, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Info struct {
			Title string `json:"title"`
		} `json:"info"`
		BasePath    string                     `json:"basePath"`
		Paths       map[string]json.RawMessage `json:"paths"`
		Definitions map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal([]byte(doc), &spec); err != nil {
		t.Fatalf("swagger doc is not valid JSON (template not rendered?): %v", err)
	}
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
