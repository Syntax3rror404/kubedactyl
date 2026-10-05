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
	if n.Kind == 0 {
		return nil, nil
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
		s := strings.TrimSpace(n.Value)
		if s == "" || s == "[]" || s == "{}" || s == "null" {
			return nil, nil
		}
		return decodeDocument([]byte(s))
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return nil, nil
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
	var done []string
	switch cfg.Done.Kind {
	case yaml.ScalarNode:
		if cfg.Done.Value != "" {
			done = []string{cfg.Done.Value}
		}
	case yaml.SequenceNode:
		for _, c := range cfg.Done.Content {
			if c.Value != "" {
				done = append(done, c.Value)
			}
		}
	}
	return done, cfg.StripAnsi, nil
}

// parseConfigFiles reads config.files: {"<file>": {"parser": "…", "find": {"<key>": value}}}.
func parseConfigFiles(n *yaml.Node) ([]v1alpha1.ConfigFile, error) {
	node, err := decodeEmbedded(n)
	if err != nil {
		return nil, fmt.Errorf("invalid config.files: %w", err)
	}
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, nil
	}
	var files []v1alpha1.ConfigFile
	for i := 0; i+1 < len(node.Content); i += 2 {
		name, body := node.Content[i].Value, node.Content[i+1]
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
	if find.Kind != yaml.MappingNode {
		return nil
	}
	var out []v1alpha1.ConfigReplace
	for j := 0; j+1 < len(find.Content); j += 2 {
		key, val := find.Content[j].Value, find.Content[j+1]
		if val.Kind != yaml.MappingNode {
			out = append(
				out,
				v1alpha1.ConfigReplace{Match: key, ReplaceWith: scalarString(val), ValueType: valueType(val)},
			)
			continue
		}
		for k := 0; k+1 < len(val.Content); k += 2 {
			out = append(out, v1alpha1.ConfigReplace{
				Match: key, IfValue: val.Content[k].Value,
				ReplaceWith: scalarString(val.Content[k+1]), ValueType: valueType(val.Content[k+1]),
			})
		}
	}
	return out
}
