package egg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// decodeDocument parses JSON or YAML into an order preserving node tree.
// JSON is decoded separately because escapes like "\/" are not valid YAML.
func decodeDocument(data []byte) (*yaml.Node, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		dec.UseNumber()
		return jsonToNode(dec)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		return doc.Content[0], nil
	}
	return nil, errors.New("empty document")
}

func jsonToNode(dec *json.Decoder) (*yaml.Node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		if v == '{' {
			return jsonObjectToNode(dec)
		}
		if v == '[' {
			return jsonArrayToNode(dec)
		}
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}, nil
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(v.String(), ".eE") {
			tag = "!!float"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: v.String()}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(v)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}

// jsonObjectToNode reads the members of an object (after its "{"), keeping their order.
func jsonObjectToNode(dec *json.Decoder) (*yaml.Node, error) {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := keyTok.(string)
		val, err := jsonToNode(dec)
		if err != nil {
			return nil, err
		}
		n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
	}
	_, err := dec.Token() // closing }
	return n, err
}

// jsonArrayToNode reads the elements of an array (after its "[").
func jsonArrayToNode(dec *json.Decoder) (*yaml.Node, error) {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for dec.More() {
		val, err := jsonToNode(dec)
		if err != nil {
			return nil, err
		}
		n.Content = append(n.Content, val)
	}
	_, err := dec.Token() // closing ]
	return n, err
}

// stringList accepts a list of strings; PHP exports empty lists as "{}".
func stringList(n *yaml.Node) []string {
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}
	var out []string
	for _, c := range n.Content {
		if c.Kind == yaml.ScalarNode && c.Value != "" {
			out = append(out, c.Value)
		}
	}
	return out
}

func mappingPairs(n *yaml.Node) [][2]string {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	var out [][2]string
	for i := 0; i+1 < len(n.Content); i += 2 {
		out = append(out, [2]string{n.Content[i].Value, n.Content[i+1].Value})
	}
	return out
}

func scalarString(n *yaml.Node) string {
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
		return ""
	}
	return n.Value
}

// valueType returns the JSON type of a scalar; strings are the default and left empty.
func valueType(n *yaml.Node) string {
	switch n.Tag {
	case "!!bool":
		return "boolean"
	case "!!int", "!!float":
		return "number"
	}
	return ""
}

func rulesString(n *yaml.Node) string {
	switch n.Kind {
	case yaml.ScalarNode:
		return n.Value
	case yaml.SequenceNode:
		var parts []string
		for _, c := range n.Content {
			parts = append(parts, c.Value)
		}
		return strings.Join(parts, "|")
	}
	return ""
}
