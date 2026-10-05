// Package egg parses Pterodactyl (PTDL_v1/v2) and Pelican (PLCN_v1-v3) egg files.
package egg

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
)

// ReservedVariables cannot be defined by eggs because they are set by the system.
var ReservedVariables = map[string]bool{
	"SERVER_MEMORY": true, "SERVER_IP": true, "SERVER_PORT": true, "ENV": true, "HOME": true,
	"USER": true, "STARTUP": true, "SERVER_UUID": true, "UUID": true, "P_SERVER_UUID": true,
	"P_SERVER_LOCATION": true, "P_SERVER_ALLOCATION_LIMIT": true, "TZ": true,
}

type rawEgg struct {
	Meta struct {
		Version   string `yaml:"version"`
		UpdateURL string `yaml:"update_url"`
	} `yaml:"meta"`
	ExportedAt      string    `yaml:"exported_at"`
	Name            string    `yaml:"name"`
	Author          string    `yaml:"author"`
	UUID            string    `yaml:"uuid"`
	Tags            yaml.Node `yaml:"tags"`
	Description     string    `yaml:"description"`
	Icon            string    `yaml:"icon"`
	Features        yaml.Node `yaml:"features"`
	DockerImages    yaml.Node `yaml:"docker_images"`
	Image           string    `yaml:"image"`
	Images          yaml.Node `yaml:"images"`
	FileDenylist    yaml.Node `yaml:"file_denylist"`
	Startup         string    `yaml:"startup"`
	StartupCommands yaml.Node `yaml:"startup_commands"`
	Config          struct {
		Files   yaml.Node `yaml:"files"`
		Startup yaml.Node `yaml:"startup"`
		Stop    string    `yaml:"stop"`
	} `yaml:"config"`
	Scripts struct {
		Installation struct {
			Script     string `yaml:"script"`
			Container  string `yaml:"container"`
			Entrypoint string `yaml:"entrypoint"`
		} `yaml:"installation"`
	} `yaml:"scripts"`
	Variables []struct {
		Name         string    `yaml:"name"`
		Description  string    `yaml:"description"`
		EnvVariable  string    `yaml:"env_variable"`
		DefaultValue yaml.Node `yaml:"default_value"`
		UserViewable bool      `yaml:"user_viewable"`
		UserEditable bool      `yaml:"user_editable"`
		Rules        yaml.Node `yaml:"rules"`
		FieldType    string    `yaml:"field_type"`
		Sort         *int      `yaml:"sort"`
	} `yaml:"variables"`
}

var supportedFormats = map[string]bool{
	"PTDL_v1": true, "PTDL_v2": true, "PLCN_v1": true, "PLCN_v2": true, "PLCN_v3": true,
}

// Parse converts an egg file (JSON or YAML, Pterodactyl or Pelican format) into a normalized
// EggSpec. The formats differ in details that are smoothed out here: docker_images as map or
// list, startup vs. startup_commands, config.* as JSON strings or objects, rules as string or list.
func Parse(data []byte) (*v1alpha1.EggSpec, error) {
	raw, err := decodeEgg(data)
	if err != nil {
		return nil, err
	}
	spec := &v1alpha1.EggSpec{
		DisplayName:  raw.Name,
		Author:       raw.Author,
		Description:  strings.TrimSpace(raw.Description),
		Features:     stringList(&raw.Features),
		Tags:         stringList(&raw.Tags),
		FileDenylist: stringList(&raw.FileDenylist),
		Stop:         raw.Config.Stop,
		Install: v1alpha1.InstallScript{
			// Install scripts run with LF line endings (egg files often contain CRLF).
			Script:     strings.ReplaceAll(raw.Scripts.Installation.Script, "\r\n", "\n"),
			Container:  raw.Scripts.Installation.Container,
			Entrypoint: raw.Scripts.Installation.Entrypoint,
		},
		Source: v1alpha1.EggSource{
			Format: raw.Meta.Version, UpdateURL: raw.Meta.UpdateURL, UUID: raw.UUID,
			ExportedAt: exportedAt(raw.ExportedAt),
		},
		Variables: parseVariables(raw),
	}
	if spec.DockerImages = parseDockerImages(raw); len(spec.DockerImages) == 0 {
		return nil, errors.New("egg has no docker images")
	}
	if spec.Startup = parseStartup(raw); spec.Startup == "" {
		return nil, errors.New("egg has no startup command")
	}
	// Pelican eggs embed an icon as data URI; keep it when it is reasonably small.
	if strings.HasPrefix(raw.Icon, "data:image/") && len(raw.Icon) <= maxIconSize {
		spec.Icon = raw.Icon
	}
	if spec.StartupDone, spec.StripAnsi, err = parseStartupConfig(&raw.Config.Startup); err != nil {
		return nil, err
	}
	if spec.ConfigFiles, err = parseConfigFiles(&raw.Config.Files); err != nil {
		return nil, err
	}
	return spec, nil
}

// decodeEgg reads the document and checks the fields every format has.
func decodeEgg(data []byte) (*rawEgg, error) {
	var raw rawEgg
	root, err := decodeDocument(data)
	if err != nil {
		return nil, fmt.Errorf("invalid egg file: %w", err)
	}
	if err := root.Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid egg file: %w", err)
	}
	switch {
	case raw.Meta.Version == "":
		return nil, errors.New("invalid egg file: meta.version is missing")
	case !supportedFormats[raw.Meta.Version]:
		return nil, fmt.Errorf("unsupported egg format %q", raw.Meta.Version)
	case raw.Name == "":
		return nil, errors.New("invalid egg file: name is missing")
	}
	return &raw, nil
}

// parseDockerImages reads docker_images (an ordered map in v2/PLCN, sometimes a plain list) or
// the image/images fields of PTDL_v1.
func parseDockerImages(raw *rawEgg) []v1alpha1.DockerImage {
	var images []v1alpha1.DockerImage
	for _, kv := range mappingPairs(&raw.DockerImages) {
		images = append(images, v1alpha1.DockerImage{Name: kv[0], Image: kv[1]})
	}
	for _, img := range stringList(&raw.DockerImages) {
		images = append(images, v1alpha1.DockerImage{Name: img, Image: img})
	}
	if len(images) > 0 {
		return images
	}
	legacy := stringList(&raw.Images)
	if raw.Image != "" && !strings.HasPrefix(raw.Image, "data:") {
		legacy = append([]string{raw.Image}, legacy...)
	}
	for _, img := range legacy {
		images = append(images, v1alpha1.DockerImage{Name: img, Image: img})
	}
	return images
}

// parseStartup returns the startup command: a string (PTDL) or the first of the named
// startup_commands (PLCN_v3).
func parseStartup(raw *rawEgg) string {
	if raw.Startup != "" {
		return raw.Startup
	}
	if pairs := mappingPairs(&raw.StartupCommands); len(pairs) > 0 {
		return pairs[0][1]
	}
	return ""
}

// parseVariables converts the variables, ordered by their sort field (file order otherwise).
func parseVariables(raw *rawEgg) []v1alpha1.EggVariable {
	type sortable struct {
		v    v1alpha1.EggVariable
		sort int
	}
	var vars []sortable
	for i, v := range raw.Variables {
		if v.EnvVariable == "" {
			continue
		}
		order := i
		if v.Sort != nil {
			order = *v.Sort
		}
		vars = append(vars, sortable{v1alpha1.EggVariable{
			Name:         v.Name,
			Description:  strings.TrimSpace(strings.ReplaceAll(v.Description, "\r\n", "\n")),
			EnvVariable:  v.EnvVariable,
			DefaultValue: scalarString(&v.DefaultValue),
			UserViewable: v.UserViewable,
			UserEditable: v.UserEditable,
			Rules:        rulesString(&v.Rules),
			FieldType:    cmp.Or(v.FieldType, "text"),
		}, order})
	}
	sort.SliceStable(vars, func(i, j int) bool { return vars[i].sort < vars[j].sort })
	out := make([]v1alpha1.EggVariable, 0, len(vars))
	for _, v := range vars {
		out = append(out, v.v)
	}
	return out
}

// maxIconSize keeps Egg objects well below the etcd object size limit.
const maxIconSize = 256 << 10

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug converts a display name into a valid Kubernetes object name.
func Slug(name string) string {
	s := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 50 {
		s = strings.TrimRight(s[:50], "-")
	}
	if s == "" {
		s = "egg"
	}
	return s
}

// exportedAt parses the export time of an egg file (RFC 3339, as Pterodactyl and Pelican write it); an
// invalid or missing one is left out.
func exportedAt(s string) *metav1.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &metav1.Time{Time: t}
}
