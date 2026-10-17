package egg

import (
	"cmp"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"app/api/v1alpha1"
)

// decodeEmbedded returns the node itself, or the parsed document when the node is a
// JSON string (PTDL exports store config.* as JSON encoded strings).
func decodeEmbedded(n *yaml.Node) (*yaml.Node, error) {
	if n.Kind == 0 || n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return nil, nil
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
		s := strings.TrimSpace(n.Value)
		if s == "" || s == "[]" || s == "{}" || s == "null" {
			return nil, nil
		}
		return decodeDocument([]byte(s))
	}
	return n, nil
}

func parseStartupConfig(n *yaml.Node) ([]string, bool, error) {
	node, err := decodeEmbedded(n)
	if err != nil || node == nil {
		return nil, false, err
	}
	var cfg struct {
		Done      yaml.Node `yaml:"done"`
		StripAnsi bool      `yaml:"strip_ansi"`
	}
	if err := node.Decode(&cfg); err != nil {
		return nil, false, fmt.Errorf("invalid config.startup: %w", err)
	}
	// done is one text or a list of them.
	done := stringList(&cfg.Done)
	if cfg.Done.Kind == yaml.ScalarNode && cfg.Done.Value != "" {
		done = []string{cfg.Done.Value}
	}
	return done, cfg.StripAnsi, nil
}

// parseConfigFiles reads config.files: {"<file>": {"parser": "…", "find": {"<key>": value}}}.
func parseConfigFiles(n *yaml.Node) ([]v1alpha1.ConfigFile, error) {
	node, err := decodeEmbedded(n)
	if err != nil {
		return nil, fmt.Errorf("invalid config.files: %w", err)
	}
	var files []v1alpha1.ConfigFile
	for name, body := range entries(node) {
		var cfg struct {
			Parser string    `yaml:"parser"`
			Find   yaml.Node `yaml:"find"`
		}
		if err := body.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("invalid config.files[%s]: %w", name, err)
		}
		cf := v1alpha1.ConfigFile{
			File:    name,
			Parser:  strings.ToLower(cmp.Or(cfg.Parser, "file")),
			Replace: parseFind(&cfg.Find),
		}
		if cf.Parser == "yml" {
			cf.Parser = "yaml"
		}
		files = append(files, cf)
	}
	return files, nil
}

// parseFind converts the "find" map: "key": value is a plain replacement,
// "key": {"<if_value>": "<replace_with>", …} one conditional replacement per entry.
func parseFind(find *yaml.Node) []v1alpha1.ConfigReplace {
	var out []v1alpha1.ConfigReplace
	for key, val := range entries(find) {
		if val.Kind != yaml.MappingNode {
			out = append(out, replacement(key, "", val))
			continue
		}
		for ifValue, with := range entries(val) {
			out = append(out, replacement(key, ifValue, with))
		}
	}
	return out
}

func replacement(match, ifValue string, with *yaml.Node) v1alpha1.ConfigReplace {
	return v1alpha1.ConfigReplace{
		Match: match, IfValue: ifValue, ReplaceWith: scalarString(with), ValueType: valueType(with),
	}
}
