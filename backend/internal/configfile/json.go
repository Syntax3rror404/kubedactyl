package configfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/icza/dyno"
	"go.yaml.in/yaml/v3"
)

func (p *fileParser) parseJSON(content []byte) ([]byte, error) {
	if len(bytes.TrimSpace(content)) == 0 {
		content = []byte("{}")
	}
	data, err := p.iterateOverJSON(content)
	if err != nil {
		return nil, err
	}
	return data.BytesIndent("", "    "), nil
}

func (p *fileParser) parseYaml(content []byte) ([]byte, error) {
	i := make(map[string]any)
	if err := yaml.Unmarshal(content, &i); err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(dyno.ConvertMapI2MapS(i))
	if err != nil {
		return nil, err
	}
	data, err := p.iterateOverJSON(jsonBytes)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(data.Data())
}

// iterateOverJSON applies all replacements; "a.*.b" sets b on every child of a.
func (p *fileParser) iterateOverJSON(data []byte) (*gabs.Container, error) {
	parsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}
	for _, r := range p.replace {
		value := p.lookup(r)
		if strings.Contains(r.Match, ".*") {
			parts := strings.SplitN(r.Match, ".*", 2)
			for _, child := range parsed.Path(strings.Trim(parts[0], ".")).Children() {
				if err := r.setAtPathway(child, strings.Trim(parts[1], "."), value); err != nil {
					if errors.Is(err, gabs.ErrNotFound) {
						continue
					}
					return nil, fmt.Errorf("failed to set config value of array child: %w", err)
				}
			}
			continue
		}
		if err := r.setAtPathway(parsed, r.Match, value); err != nil {
			if errors.Is(err, gabs.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("unable to set config value at %s: %w", r.Match, err)
		}
	}
	return parsed, nil
}

var checkForArrayElement = regexp.MustCompile(`^([^\[\]]+)\[([\d]+)](\..+)?$`)

// setValueAtPath sets a value, supporting "list[0]" and "list[0].key" paths.
func setValueAtPath(c *gabs.Container, path string, value any) error {
	matches := checkForArrayElement.FindStringSubmatch(path)
	if len(matches) < 3 {
		_, err := c.SetP(value, path)
		return err
	}
	i, _ := strconv.Atoi(matches[2])
	ct, err := c.ArrayElementP(i, matches[1])
	if err != nil {
		if i != 0 || (!errors.Is(err, gabs.ErrNotArray) && !errors.Is(err, gabs.ErrNotFound)) {
			return fmt.Errorf("error while parsing array element at path: %w", err)
		}
		t := make([]any, 1)
		if matches[3] != "" {
			t = []any{map[string]any{}}
		}
		if _, err = c.SetP(t, matches[1]); err != nil {
			return fmt.Errorf("failed to create empty array for missing element: %w", err)
		}
		if ct, err = c.ArrayElementP(0, matches[1]); err != nil {
			return fmt.Errorf("failed to find array element at path: %w", err)
		}
	}
	if matches[3] != "" {
		_, err = ct.SetP(value, strings.TrimPrefix(matches[3], "."))
	} else {
		_, err = ct.Set(value)
	}
	return err
}

func (r Replacement) setAtPathway(c *gabs.Container, path, value string) error {
	if r.IfValue == "" {
		return setValueAtPath(c, path, r.keyValue(value))
	}
	if strings.HasPrefix(r.IfValue, "regex:") {
		if !c.ExistsP(path) {
			return gabs.ErrNotFound
		}
		re, err := regexp.Compile(strings.TrimPrefix(r.IfValue, "regex:"))
		if err != nil {
			return nil // invalid regex: skip the value
		}
		v := strings.Trim(c.Path(path).String(), "\"")
		if re.MatchString(v) {
			return setValueAtPath(c, path, re.ReplaceAllString(v, value))
		}
		return nil
	}
	if c.ExistsP(path) {
		current := strings.Trim(c.Path(path).String(), "\"")
		if current != r.IfValue {
			return nil
		}
	}
	return setValueAtPath(c, path, r.keyValue(value))
}
