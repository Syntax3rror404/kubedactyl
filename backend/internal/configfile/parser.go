// Package configfile modifies game server configuration files before a server starts.
//
// The logic is a port of the Pterodactyl Wings parser package
// (https://github.com/pterodactyl/wings/tree/develop/parser, MIT License,
// Copyright (c) 2018 - 2021 Dane Everitt and Contributors), adapted to work on
// in-memory file contents instead of the local filesystem.
//
// The format functions stay close to the upstream code on purpose (easier to compare with upstream
// fixes), even where they are longer than the rest of this code base.
//
// Deliberate deviations from upstream (both are bugs there):
//   - "regex:" if_value on json/yaml requires the key to exist (upstream inverted the check).
//   - A plain if_value on json/yaml compares the value at the path (upstream compared the whole document).
package configfile

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

// MaxFileSize caps how large a configuration file we attempt to parse. The panel parses the whole file in
// its own memory (a JSON or YAML tree takes about 20 times the file size), and configuration files of games
// are a few kilobytes.
const MaxFileSize = 1 << 20

// ValueType is the JSON type of a replacement value as written in the egg.
type ValueType string

const (
	TypeString  ValueType = "string"
	TypeNumber  ValueType = "number"
	TypeBoolean ValueType = "boolean"
)

// Replacement is one find/replace instruction.
type Replacement struct {
	Match   string
	IfValue string
	// Value has panel placeholders ({{server.*}}, {{env.*}}) already substituted.
	Value string
	Type  ValueType
}

// Config holds daemon level values available as {{config.*}} placeholders.
type Config struct {
	Docker struct {
		Interface string `json:"interface"`
		Network   struct {
			Interface string `json:"interface"`
		} `json:"network"`
	} `json:"docker"`
}

// NewConfig returns a Config where {{config.docker.interface}} resolves to iface.
func NewConfig(iface string) Config {
	var c Config
	c.Docker.Interface = iface
	c.Docker.Network.Interface = iface
	return c
}

// Apply runs the parser on content and returns the new content. When exists is false
// the file is treated as empty; the "file" parser skips missing files (write == false).
func Apply(
	parser string, content []byte, exists bool, reps []Replacement, cfg Config,
) (out []byte, write bool, err error) {
	if len(content) > MaxFileSize {
		return nil, false, fmt.Errorf("refusing to parse configuration file larger than %d bytes", MaxFileSize)
	}
	p := &fileParser{replace: reps}
	if p.configuration, err = json.Marshal(cfg); err != nil {
		return nil, false, err
	}
	switch strings.ToLower(parser) {
	case "file":
		if !exists {
			return nil, false, nil
		}
		out, err = p.parseText(content)
	case "properties":
		out, err = p.parseProperties(content)
	case "yaml", "yml":
		out, err = p.parseYaml(content)
	case "json":
		out, err = p.parseJSON(content)
	case "ini":
		out, err = p.parseIni(content)
	case "xml":
		out, err = p.parseXML(content)
	default:
		return nil, false, fmt.Errorf("unknown parser %q", parser)
	}
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

type fileParser struct {
	replace       []Replacement
	configuration []byte
}

// {{config.x.y}} lookups against the daemon configuration.
var configMatchRegex = regexp.MustCompile(`{{\s?config\.([\w.-]+)\s?}}`)

// lookup resolves {{config.*}} placeholders inside a string replacement value.
func (p *fileParser) lookup(r Replacement) string {
	if r.Type != TypeString && r.Type != "" || !configMatchRegex.MatchString(r.Value) {
		return r.Value
	}
	huntPath := configMatchRegex.ReplaceAllString(configMatchRegex.FindString(r.Value), "$1")
	var path []string
	for _, v := range strings.Split(huntPath, ".") {
		path = append(path, strcase.ToSnake(v))
	}
	var cfg map[string]any
	_ = json.Unmarshal(p.configuration, &cfg)
	var cur any = cfg
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return r.Value
		}
		if cur, ok = m[k]; !ok {
			// Keep the placeholder so the problem is visible in the file.
			return r.Value
		}
	}
	switch v := cur.(type) {
	case string:
		return configMatchRegex.ReplaceAllString(r.Value, v)
	case float64, bool:
		return configMatchRegex.ReplaceAllString(r.Value, fmt.Sprint(v))
	}
	return r.Value
}

// keyValue converts the value into the type that is written into json/yaml documents.
func (r Replacement) keyValue(value string) any {
	if r.Type == TypeBoolean {
		v, _ := strconv.ParseBool(value)
		return v
	}
	if v, err := strconv.Atoi(value); err == nil {
		return v
	}
	return value
}
