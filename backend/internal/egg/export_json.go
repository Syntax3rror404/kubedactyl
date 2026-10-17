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
	return writeJSONList(buf, "{}", len(n.Content)/2, depth, func(i int) error {
		key, _ := json.Marshal(n.Content[2*i].Value)
		buf.Write(key)
		buf.WriteString(": ")
		return writeJSON(buf, n.Content[2*i+1], depth+1)
	})
}

func writeJSONSequence(buf *bytes.Buffer, n *yaml.Node, depth int) error {
	return writeJSONList(buf, "[]", len(n.Content), depth, func(i int) error {
		return writeJSON(buf, n.Content[i], depth+1)
	})
}

// writeJSONList writes count items between the brackets, one per line; an empty list stays on
// one line ("{}", "[]").
func writeJSONList(buf *bytes.Buffer, brackets string, count, depth int, item func(i int) error) error {
	if count == 0 {
		buf.WriteString(brackets)
		return nil
	}
	buf.WriteByte(brackets[0])
	for i := range count {
		if i > 0 {
			buf.WriteByte(',')
		}
		newline(buf, depth+1)
		if err := item(i); err != nil {
			return err
		}
	}
	newline(buf, depth)
	buf.WriteByte(brackets[1])
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
