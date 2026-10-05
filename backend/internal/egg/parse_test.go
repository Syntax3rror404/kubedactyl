package egg

import (
	"slices"
	"strings"
	"testing"

	"app/api/v1alpha1"
	"app/internal/testutil"
)

func TestParsePaper(t *testing.T) {
	for _, url := range []string{testutil.PaperPTDL, testutil.PaperPLCN} {
		t.Run(url, func(t *testing.T) {
			spec, err := Parse(testutil.Download(t, url))
			if err != nil {
				t.Fatal(err)
			}
			checkPaper(t, spec)
		})
	}
}

// checkPaper checks what every version of the upstream Paper egg has (the files change upstream,
// so exact versions and counts are not compared).
func checkPaper(t *testing.T, spec *v1alpha1.EggSpec) {
	t.Helper()
	if spec.DisplayName != "Paper" || spec.Stop != "stop" {
		t.Errorf("name = %q stop = %q", spec.DisplayName, spec.Stop)
	}
	// The images keep the order of the file (newest Java first upstream).
	if len(spec.DockerImages) < 2 || !strings.HasPrefix(spec.DockerImages[0].Name, "Java ") {
		t.Errorf("docker images = %+v", spec.DockerImages)
	}
	if len(spec.StartupDone) == 0 || !strings.Contains(spec.StartupDone[0], "For help") {
		t.Errorf("done = %q", spec.StartupDone)
	}
	files := spec.ConfigFiles
	if len(files) != 1 || files[0].Parser != "properties" || len(files[0].Replace) == 0 {
		t.Errorf("config files = %+v", spec.ConfigFiles)
	}
	if spec.Install.Container == "" || spec.Install.Entrypoint == "" || spec.Install.Script == "" {
		t.Errorf("install = %+v", spec.Install)
	}
	jar := func(v v1alpha1.EggVariable) bool { return v.EnvVariable == "SERVER_JARFILE" }
	if !slices.ContainsFunc(spec.Variables, jar) {
		t.Errorf("variables = %+v", spec.Variables)
	}
	if spec.Source.ExportedAt == nil || spec.Source.ExportedAt.IsZero() {
		t.Errorf("exported at = %v", spec.Source.ExportedAt)
	}
}
