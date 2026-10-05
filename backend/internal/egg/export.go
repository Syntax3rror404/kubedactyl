package egg

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"

	"app/api/v1alpha1"
)

// Export formats. PLCN_v3 is the Pelican format (YAML or JSON, same content); PTDL_v2 is the
// Pterodactyl format (JSON only) for importing an egg back into Pterodactyl.
const (
	FormatYAML = "yaml"
	FormatJSON = "json"
	FormatPTDL = "ptdl"
)

// namespaceEggs derives stable UUIDs for eggs imported without one (older PTDL files).
var namespaceEggs = uuid.MustParse("6f1c1d0e-3b8e-4c55-9d7c-5b0f2c6e8a41")

// UUIDFor returns the egg's UUID, or a stable one derived from its object name.
func UUIDFor(name string, spec *v1alpha1.EggSpec) string {
	if spec.Source.UUID != "" {
		return spec.Source.UUID
	}
	return uuid.NewSHA1(namespaceEggs, []byte(name)).String()
}

// NewUUID returns a random UUID for new eggs.
func NewUUID() string { return uuid.NewString() }

// Export writes the egg like the Pelican (PLCN_v3) or Pterodactyl (PTDL_v2) exporter does, with the
// same key order. It returns the content and the file name.
func Export(name string, spec *v1alpha1.EggSpec, format string, now time.Time) ([]byte, string, error) {
	if format != FormatYAML && format != FormatJSON && format != FormatPTDL {
		return nil, "", fmt.Errorf("unknown format %q (yaml, json or ptdl)", format)
	}
	doc, err := exportDocument(name, spec, format, now)
	if err != nil {
		return nil, "", err
	}
	file := "egg-" + Slug(spec.DisplayName)
	var buf bytes.Buffer
	if format == FormatYAML {
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		err := enc.Encode(doc.node())
		return buf.Bytes(), file + ".yaml", err
	}
	if err := writeJSON(&buf, doc.node(), 0); err != nil {
		return nil, "", err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), file + ".json", nil
}

// exportDocument builds the egg document in the key order of the respective exporter.
func exportDocument(name string, spec *v1alpha1.EggSpec, format string, now time.Time) (*mapping, error) {
	ptdl := format == FormatPTDL
	doc := &mapping{}
	if ptdl {
		doc.add("_comment", str("DO NOT EDIT: FILE GENERATED AUTOMATICALLY BY PTERODACTYL PANEL - PTERODACTYL.IO"))
		doc.add(
			"meta",
			(&mapping{}).add("version", str("PTDL_v2")).add("update_url", strOrNull(spec.Source.UpdateURL)).node(),
		)
	} else {
		doc.add("_comment", str("DO NOT EDIT: FILE GENERATED AUTOMATICALLY BY PANEL"))
		doc.add(
			"meta",
			(&mapping{}).add("version", str("PLCN_v3")).add("update_url", strOrNull(spec.Source.UpdateURL)).node(),
		)
	}
	doc.add("exported_at", str(now.Format("2006-01-02T15:04:05-07:00")))
	doc.add("name", str(spec.DisplayName))
	doc.add("author", str(spec.Author))
	if !ptdl {
		doc.add("uuid", str(UUIDFor(name, spec)))
	}
	doc.add("description", str(spec.Description))
	if !ptdl {
		doc.add("icon", strOrNull(spec.Icon))
		doc.add("tags", strList(spec.Tags))
	}
	doc.add("features", strList(spec.Features))
	images := &mapping{}
	for _, img := range spec.DockerImages {
		images.add(img.Name, str(img.Image))
	}
	doc.add("docker_images", images.node())
	doc.add("file_denylist", strList(spec.FileDenylist))
	if ptdl {
		doc.add("startup", str(spec.Startup))
	} else {
		doc.add("startup_commands", (&mapping{}).add("Default", str(spec.Startup)).node())
	}
	// JSON exports keep config.files/startup/logs as JSON strings, the YAML export as maps.
	config, err := exportConfig(spec, format != FormatYAML)
	if err != nil {
		return nil, err
	}
	doc.add("config", config)
	install := (&mapping{}).add("script", str(spec.Install.Script)).
		add("container", str(spec.Install.Container)).add("entrypoint", str(spec.Install.Entrypoint))
	doc.add("scripts", (&mapping{}).add("installation", install.node()).node())
	doc.add("variables", exportVariables(spec.Variables, ptdl))
	return doc, nil
}

// exportConfig writes config.files, config.startup, config.logs (as objects or JSON strings) and config.stop.
func exportConfig(spec *v1alpha1.EggSpec, asStrings bool) (*yaml.Node, error) {
	config := &mapping{}
	parts := []struct {
		key  string
		node *yaml.Node
	}{{"files", configFilesNode(spec.ConfigFiles)}, {"startup", startupNode(spec)}, {"logs", (&mapping{}).node()}}
	for _, part := range parts {
		if !asStrings {
			config.add(part.key, part.node)
			continue
		}
		text, err := jsonString(part.node)
		if err != nil {
			return nil, err
		}
		config.add(part.key, str(text))
	}
	config.add("stop", str(spec.Stop))
	return config.node(), nil
}

// exportVariables writes the variables: rules as "a|b" and field_type for PTDL, rules as a
// list and sort for PLCN.
func exportVariables(vars []v1alpha1.EggVariable, ptdl bool) *yaml.Node {
	out := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for i, v := range vars {
		m := (&mapping{}).add("name", str(v.Name)).add("description", str(v.Description)).
			add("env_variable", str(v.EnvVariable)).add("default_value", str(v.DefaultValue)).
			add("user_viewable", boolean(v.UserViewable)).add("user_editable", boolean(v.UserEditable))
		if ptdl {
			m.add("rules", str(v.Rules)).add("field_type", str("text"))
		} else {
			m.add("rules", strList(SplitRules(v.Rules))).
				add("sort", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(i + 1)})
		}
		out.Content = append(out.Content, m.node())
	}
	return out
}

// configFilesNode builds {"file": {"parser": …, "find": {"key": value | {"if": value}}}}.
func configFilesNode(files []v1alpha1.ConfigFile) *yaml.Node {
	out := &mapping{}
	for _, f := range files {
		find := &mapping{}
		conditional := map[string]*mapping{}
		for _, r := range f.Replace {
			if r.IfValue == "" {
				find.add(r.Match, typed(r.ReplaceWith, r.ValueType))
				continue
			}
			m, ok := conditional[r.Match]
			if !ok {
				m = &mapping{}
				conditional[r.Match] = m
				find.add(r.Match, nil) // placeholder, keeps the key order
			}
			m.add(r.IfValue, typed(r.ReplaceWith, r.ValueType))
		}
		for i := 0; i < len(find.pairs); i++ {
			if find.pairs[i].value == nil {
				find.pairs[i].value = conditional[find.pairs[i].key].node()
			}
		}
		out.add(f.File, (&mapping{}).add("parser", str(f.Parser)).add("find", find.node()).node())
	}
	return out.node()
}

func startupNode(spec *v1alpha1.EggSpec) *yaml.Node {
	m := &mapping{}
	switch len(spec.StartupDone) {
	case 0:
	case 1:
		m.add("done", str(spec.StartupDone[0]))
	default:
		m.add("done", strList(spec.StartupDone))
	}
	if spec.StripAnsi {
		m.add("strip_ansi", boolean(true))
	}
	return m.node()
}

// typed returns the value with its original JSON type (config files keep numbers and booleans).
func typed(value, valueType string) *yaml.Node {
	switch valueType {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			tag := "!!float"
			if _, err := strconv.ParseInt(value, 10, 64); err == nil {
				tag = "!!int"
			}
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
		}
	case "boolean":
		if value == "true" || value == "false" {
			return boolean(value == "true")
		}
	}
	return str(value)
}

// mapping is an ordered map that becomes a YAML mapping node.
type mapping struct {
	pairs []struct {
		key   string
		value *yaml.Node
	}
}

func (m *mapping) add(key string, value *yaml.Node) *mapping {
	m.pairs = append(m.pairs, struct {
		key   string
		value *yaml.Node
	}{key, value})
	return m
}

func (m *mapping) node() *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range m.pairs {
		n.Content = append(n.Content, str(p.key), p.value)
	}
	return n
}

func str(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if strings.Contains(s, "\n") {
		n.Style = yaml.LiteralStyle
	}
	return n
}

func strOrNull(s string) *yaml.Node {
	if s == "" {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}
	return str(s)
}

func boolean(b bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(b)}
}

func strList(list []string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, s := range list {
		n.Content = append(n.Content, str(s))
	}
	return n
}
