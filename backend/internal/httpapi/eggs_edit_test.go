package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/egg"
)

func TestEggEditing(t *testing.T) {
	h := newHarness(t)
	admin := h.login("admin")
	spec := v1alpha1.EggSpec{
		DisplayName:  "Sample Game",
		Author:       "dev@example.com",
		Startup:      "./start.sh",
		DockerImages: []v1alpha1.DockerImage{{Name: "Debian", Image: "ghcr.io/x/debian"}},
		Variables: []v1alpha1.EggVariable{
			{Name: "Map", EnvVariable: "MAP", DefaultValue: "world", Rules: "required|string"},
		},
	}

	bad := spec
	bad.Author = ""
	bad.Variables = []v1alpha1.EggVariable{{Name: "Port", EnvVariable: "SERVER_PORT"}}
	code, body := h.do("POST", "/api/eggs", admin, bad)
	expect(t, "invalid egg", code, 422, body)
	if !strings.Contains(body, `"author"`) || !strings.Contains(body, `"variables.0.envVariable"`) {
		t.Errorf("field errors missing: %s", body)
	}

	var created v1alpha1.Egg
	for i, want := range []string{"sample-game", "sample-game-2"} {
		code, body = h.do("POST", "/api/eggs", admin, spec)
		expect(t, "create egg", code, 201, body)
		_ = json.Unmarshal([]byte(body), &created)
		if created.Name != want || created.Spec.Source.UUID == "" || created.Spec.Source.EditedAt == nil {
			t.Errorf("create #%d: name %q, source %+v", i+1, created.Name, created.Spec.Source)
		}
	}

	var before v1alpha1.Egg
	_ = h.client.Get(t.Context(), client.ObjectKey{Namespace: sysNS, Name: "sample-game"}, &before)
	edited := spec
	edited.Startup = "./start.sh --edited"
	edited.Source.UpdateURL = "https://example.com/egg.yaml"
	code, body = h.do("PUT", "/api/eggs/sample-game", admin, edited)
	expect(t, "update egg", code, 200, body)
	var after v1alpha1.Egg
	_ = json.Unmarshal([]byte(body), &after)
	if after.Spec.Startup != edited.Startup || after.Spec.Source.UUID != before.Spec.Source.UUID ||
		after.Spec.Source.UpdateURL != edited.Source.UpdateURL {
		t.Errorf("update: %+v (uuid before %s)", after.Spec, before.Spec.Source.UUID)
	}

	for _, format := range []string{"yaml", "json", "ptdl"} {
		req := httptest.NewRequest("GET", "/api/eggs/sample-game/export?download=true&format="+format, nil)
		req.Header.Set("Authorization", "Bearer "+admin)
		w := httptest.NewRecorder()
		h.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Disposition"), "egg-sample-game.") {
			t.Fatalf("export %s: %d %s", format, w.Code, w.Header())
		}
		back, err := egg.Parse(w.Body.Bytes())
		if err != nil || back.Startup != edited.Startup || back.Variables[0].EnvVariable != "MAP" {
			t.Errorf("export %s does not import again: %v", format, err)
		}
	}
	code, body = h.do("POST", "/api/eggs/sample-game-2/update-from-url", admin, nil)
	expect(t, "update without URL", code, 409, body)
}
