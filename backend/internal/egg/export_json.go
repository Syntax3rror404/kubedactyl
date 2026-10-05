package egg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// jsonString encodes a node as pretty JSON text (PTDL stores config.* like this).
func jsonString(n *yaml.Node) (string, error) {
	var buf bytes.Buffer
	if len(n.Content) == 0 && n.Kind == yaml.MappingNode {
		return "{}", nil
	}
	err := writeJSON(&buf, n, 0)
	return buf.String(), err
}

// writeJSON writes a node tree as JSON indented with four spaces (like PHP's JSON_PRETTY_PRINT),
// keeping the key order.
func writeJSON(buf *bytes.Buffer, n *yaml.Node, depth int) error {
	switch n.Kind {
	case yaml.MappingNode:
		return writeJSONMapping(buf, n, depth)
	case yaml.SequenceNode:
		return writeJSONSequence(buf, n, depth)
	case yaml.ScalarNode:
		return writeJSONScalar(buf, n)
	}
	return fmt.Errorf("unexpected YAML node kind %d", n.Kind)
}

func newline(buf *bytes.Buffer, depth int) { buf.WriteString("\n" + strings.Repeat("    ", depth)) }

func writeJSONMapping(buf *bytes.Buffer, n *yaml.Node, depth int) error {
	if len(n.Content) == 0 {
		buf.WriteString("{}")
		return nil
	}
	buf.WriteByte('{')
	for i := 0; i+1 < len(n.Content); i += 2 {
		if i > 0 {
			buf.WriteByte(',')
		}
		newline(buf, depth+1)
		key, _ := json.Marshal(n.Content[i].Value)
		buf.Write(key)
		buf.WriteString(": ")
		if err := writeJSON(buf, n.Content[i+1], depth+1); err != nil {
			return err
		}
	}
	newline(buf, depth)
	buf.WriteByte('}')
	return nil
}

func writeJSONSequence(buf *bytes.Buffer, n *yaml.Node, depth int) error {
	if len(n.Content) == 0 {
		buf.WriteString("[]")
		return nil
	}
	buf.WriteByte('[')
	for i, c := range n.Content {
		if i > 0 {
			buf.WriteByte(',')
		}
		newline(buf, depth+1)
		if err := writeJSON(buf, c, depth+1); err != nil {
			return err
		}
	}
	newline(buf, depth)
	buf.WriteByte(']')
	return nil
}

// writeJSONScalar writes null/bool/number as they are and everything else as a JSON string
// (without escaping <, > and & like json.Marshal would).
func writeJSONScalar(buf *bytes.Buffer, n *yaml.Node) error {
	switch n.Tag {
	case "!!null", "!!bool", "!!int", "!!float":
		buf.WriteString(n.Value)
		return nil
	}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(n.Value); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1) // Encode adds a newline
	return nil
}
