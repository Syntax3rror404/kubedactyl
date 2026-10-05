package egg

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/testutil"
)

// sample covers what the Paper eggs do not: conditional and typed replacements, several
// "done" strings, tags, a multi-line script.
func sample() *v1alpha1.EggSpec {
	return &v1alpha1.EggSpec{
		DisplayName: "Sample",
		Author:      "dev@example.com",
		Description: "Line one.\nLine two.",
		Tags:        []string{"test"},
		Features:    []string{"eula"},
		DockerImages: []v1alpha1.DockerImage{
			{Name: "Java 21", Image: "ghcr.io/x/java:21"},
			{Name: "Java 17", Image: "ghcr.io/x/java:17"},
		},
		Startup:     "java -jar {{SERVER_JARFILE}}",
		Stop:        "^C",
		StartupDone: []string{"Done (", "regex:^Ready$"},
		StripAnsi:   true,
		ConfigFiles: []v1alpha1.ConfigFile{{File: "config/app.json", Parser: "json", Replace: []v1alpha1.ConfigReplace{
			{Match: "port", ReplaceWith: "{{server.build.default.port}}"},
			{Match: "max", ReplaceWith: "20", ValueType: "number"},
			{Match: "online", ReplaceWith: "true", ValueType: "boolean"},
			{Match: "mode", IfValue: "old", ReplaceWith: "new"},
			{Match: "mode", IfValue: "legacy", ReplaceWith: "new"},
		}}},
		Install: v1alpha1.InstallScript{
			Script:     "#!/bin/ash\necho hi\n",
			Container:  "ghcr.io/x/installer:alpine",
			Entrypoint: "ash",
		},
		Variables: []v1alpha1.EggVariable{
			{
				Name:         "Jar",
				EnvVariable:  "SERVER_JARFILE",
				DefaultValue: "server.jar",
				UserViewable: true,
				UserEditable: true,
				Rules:        "required|regex:/^([\\w\\d._-]+)(\\.jar)$/",
				FieldType:    "text",
			},
		},
		Source: v1alpha1.EggSource{
			UpdateURL: "https://example.com/egg.yaml",
			UUID:      "5da37ef6-58da-4169-90a6-e683e1721247",
		},
	}
}

// comparable drops what an export does not carry (import metadata; for PTDL also uuid, tags, icon).
func comparable(s v1alpha1.EggSpec, ptdl bool) v1alpha1.EggSpec {
	s.Source.Format, s.Source.ImportedAt, s.Source.ImportedFrom, s.Source.EditedAt = "", metav1.Time{}, "", nil
	s.Source.ExportedAt = nil // an export always carries the time of the export
	if ptdl {
		s.Source.UUID, s.Tags, s.Icon = "", nil, ""
	}
	return s
}

func TestExportRoundTrip(t *testing.T) {
	specs := map[string]*v1alpha1.EggSpec{"sample": sample()}
	for _, url := range []string{testutil.PaperPTDL, testutil.PaperPLCN} {
		var err error
		if specs[url], err = Parse(testutil.Download(t, url)); err != nil {
			t.Fatal(err)
		}
	}
	for name, spec := range specs {
		for _, format := range []string{FormatYAML, FormatJSON, FormatPTDL} {
			checkRoundTrip(t, name, spec, format)
		}
	}
}

// checkRoundTrip exports the spec in the format, imports it again and compares both.
func checkRoundTrip(t *testing.T, name string, spec *v1alpha1.EggSpec, format string) {
	t.Helper()
	out, file, err := Export("paper", spec, format, time.Now())
	if err != nil {
		t.Fatalf("%s/%s: %v", name, format, err)
	}
	back, err := Parse(out)
	if err != nil {
		t.Fatalf("%s/%s: re-import: %v\n%s", name, format, err, out)
	}
	want := comparable(*spec, format == FormatPTDL)
	if format == FormatPTDL || spec.Source.UUID == "" {
		want.Source.UUID = back.Source.UUID // derived or dropped
	}
	if got := comparable(*back, format == FormatPTDL); !reflect.DeepEqual(got, want) {
		t.Errorf("%s/%s (%s) changed on the round trip:\n got %+v\nwant %+v", name, format, file, got, want)
	}
}

func TestExportFormats(t *testing.T) {
	spec := sample()
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	var plcn map[string]any
	out, file, _ := Export("sample", spec, FormatJSON, now)
	if err := json.Unmarshal(out, &plcn); err != nil || file != "egg-sample.json" {
		t.Fatalf("%s: %v", file, err)
	}
	meta := plcn["meta"].(map[string]any)
	if meta["version"] != "PLCN_v3" || plcn["uuid"] != spec.Source.UUID ||
		plcn["exported_at"] != "2026-09-29T12:00:00+00:00" {
		t.Errorf("PLCN header: %v %v %v", meta, plcn["uuid"], plcn["exported_at"])
	}
	cfg := plcn["config"].(map[string]any)
	if _, ok := cfg["files"].(string); !ok {
		t.Errorf("JSON export must keep config.files as a JSON string like Pelican: %T", cfg["files"])
	}
	if rules := plcn["variables"].([]any)[0].(map[string]any)["rules"]; !reflect.DeepEqual(
		rules,
		[]any{"required", "regex:/^([\\w\\d._-]+)(\\.jar)$/"},
	) {
		t.Errorf("PLCN rules must be a list: %v", rules)
	}
	if !strings.HasPrefix(string(out), "{\n    \"_comment\"") {
		t.Errorf("key order / indentation: %.60s", out)
	}

	out, file, _ = Export("sample", spec, FormatYAML, now)
	var y map[string]any
	if err := yaml.Unmarshal(out, &y); err != nil || file != "egg-sample.yaml" {
		t.Fatalf("%s: %v", file, err)
	}
	if _, ok := y["config"].(map[string]any)["files"].(map[string]any); !ok {
		t.Errorf("YAML export must write config.files as a map")
	}
	if !strings.Contains(string(out), "script: |") {
		t.Errorf("multi-line scripts should be literal blocks:\n%s", out)
	}

	var ptdl map[string]any
	out, _, _ = Export("sample", spec, FormatPTDL, now)
	if err := json.Unmarshal(out, &ptdl); err != nil {
		t.Fatal(err)
	}
	if ptdl["meta"].(map[string]any)["version"] != "PTDL_v2" || ptdl["startup"] != spec.Startup {
		t.Errorf("PTDL header/startup: %v %v", ptdl["meta"], ptdl["startup"])
	}
	for _, k := range []string{"_comment", "uuid", "tags", "icon", "startup_commands"} {
		if _, ok := ptdl[k]; ok {
			t.Errorf("PTDL_v2 has no %q", k)
		}
	}
	v := ptdl["variables"].([]any)[0].(map[string]any)
	if v["rules"] != spec.Variables[0].Rules || v["field_type"] != "text" {
		t.Errorf("PTDL variable: %v", v)
	}
}

func TestUUIDFor(t *testing.T) {
	s := &v1alpha1.EggSpec{}
	if a, b := UUIDFor("factorio", s), UUIDFor("factorio", s); a != b || a == UUIDFor("paper", s) {
		t.Errorf("derived UUIDs must be stable per egg: %s %s", a, b)
	}
}

func TestValidateSpec(t *testing.T) {
	if err := ValidateSpec(sample()); err != nil {
		t.Fatalf("valid egg rejected: %v", err)
	}
	bad := sample()
	bad.Author = "someone"
	bad.DockerImages = append(bad.DockerImages, v1alpha1.DockerImage{Name: "Java 21", Image: "ghcr.io/x/java:22"})
	bad.Variables = append(bad.Variables,
		v1alpha1.EggVariable{Name: "Port", EnvVariable: "SERVER_PORT", Rules: "required"},
		v1alpha1.EggVariable{Name: "Dash", EnvVariable: "MY-VAR"},
		v1alpha1.EggVariable{Name: "Host", EnvVariable: "hostname"},
		v1alpha1.EggVariable{Name: "Again", EnvVariable: "SERVER_JARFILE", Rules: "regex:/(/"},
		v1alpha1.EggVariable{Name: "Max", EnvVariable: "MAX", Rules: "max"},
	)
	bad.ConfigFiles = append(bad.ConfigFiles, v1alpha1.ConfigFile{File: "../etc/passwd", Parser: "toml"})
	bad.StartupDone = append(bad.StartupDone, "regex:(")
	err, _ := ValidateSpec(bad).(FieldErrors)
	got := (FieldErrors{"variables.0.envVariable": "is reserved"}).Error()
	if got != "Variable 1, environment variable: is reserved" {
		t.Errorf("message = %q", got)
	}
	for _, field := range []string{
		"author", "dockerImages.2.name", "variables.1.envVariable", "variables.2.envVariable",
		"variables.3.envVariable", "variables.4.envVariable", "variables.4.rules", "variables.5.rules",
		"configFiles.1.file", "configFiles.1.parser", "startupDone.2",
	} {
		if err[field] == "" {
			t.Errorf("expected an error for %s, got %v", field, err)
		}
	}
}
