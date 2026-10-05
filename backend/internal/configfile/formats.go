package configfile

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/magiconair/properties"
	"gopkg.in/ini.v1"
)

// parseText replaces every line starting with Match by the replacement (no {{config}} lookup).
func (p *fileParser) parseText(content []byte) ([]byte, error) {
	var b bytes.Buffer
	s := bufio.NewScanner(bytes.NewReader(content))
	s.Buffer(make([]byte, 0, 64*1024), MaxFileSize)
	for s.Scan() {
		line := s.Bytes()
		replaced := false
		for _, r := range p.replace {
			if !bytes.HasPrefix(line, []byte(r.Match)) {
				continue
			}
			b.WriteString(r.Value)
			replaced = true
		}
		if !replaced {
			b.Write(line)
		}
		b.WriteByte('\n')
	}
	return b.Bytes(), s.Err()
}

// parseProperties keeps the leading comment block and rewrites all keys as key=value.
func (p *fileParser) parseProperties(content []byte) ([]byte, error) {
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		text := scanner.Bytes()
		if len(text) > 0 && text[0] != '#' {
			break
		}
		out.Write(text)
		out.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Without expansion: "${key}" stays as written (the game reads it literally), and a file with nested
	// references cannot grow into gigabytes in the panel's memory.
	loader := properties.Loader{Encoding: properties.UTF8, DisableExpansion: true}
	props, err := loader.LoadBytes(content)
	if err != nil {
		return nil, fmt.Errorf("could not load properties file: %w", err)
	}
	for _, r := range p.replace {
		data := p.lookup(r)
		v, ok := props.Get(r.Match)
		if r.IfValue != "" && (!ok || v != r.IfValue) {
			continue
		}
		if _, _, err := props.Set(r.Match, data); err != nil {
			return nil, err
		}
	}
	for _, key := range props.Keys() {
		value, ok := props.Get(key)
		if !ok {
			continue
		}
		// Escaping non-ASCII characters is intentional: eggs rely on it (pterodactyl/panel#2308).
		out.WriteString(key + "=" + strings.Trim(strconv.QuoteToASCII(value), "\"") + "\n")
	}
	return out.Bytes(), nil
}

// parseIni supports "key" (default section), "section.key" and "[section.with.dots].key".
func (p *fileParser) parseIni(content []byte) ([]byte, error) {
	cfg, err := ini.Load(content)
	if err != nil {
		return nil, err
	}
	for _, r := range p.replace {
		var (
			path         []string
			bracketDepth int
			v            []rune
		)
		for _, c := range r.Match {
			switch c {
			case '[':
				bracketDepth++
			case ']':
				bracketDepth--
			case '.':
				if bracketDepth > 0 || len(path) == 1 {
					v = append(v, c)
					continue
				}
				path = append(path, string(v))
				v = v[:0]
			default:
				v = append(v, c)
			}
		}
		path = append(path, string(v))

		value := p.lookup(r)
		k := path[0]
		s := cfg.Section("")
		if len(path) == 2 {
			k = path[1]
			s = cfg.Section(path[0])
		}
		if s.HasKey(k) {
			s.Key(k).SetValue(value)
		} else if _, err := s.NewKey(k, value); err != nil {
			return nil, err
		}
	}
	var out bytes.Buffer
	if _, err := cfg.WriteTo(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// parseXML maps "a.b.c" to the element path ./a/b/c and creates missing elements.
func (p *fileParser) parseXML(content []byte) ([]byte, error) {
	doc := etree.NewDocument()
	if len(bytes.TrimSpace(content)) > 0 {
		if err := doc.ReadFromBytes(content); err != nil {
			return nil, err
		}
	}
	if doc.Root() == nil {
		doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
	}
	for i, r := range p.replace {
		value := p.lookup(r)
		if i == 0 && doc.Root() == nil {
			parts := strings.SplitN(r.Match, ".", 2)
			doc.SetRoot(doc.CreateElement(parts[0]))
		}
		path := "./" + strings.ReplaceAll(r.Match, ".", "/")
		if !strings.Contains(path, "*") {
			parts := strings.Split(r.Match, ".")
			element := doc.Root()
			for _, tag := range parts[1:] {
				if e := element.FindElement(tag); e == nil {
					element = element.CreateElement(tag)
				} else {
					element = e
				}
			}
		}
		for _, element := range doc.FindElements(path) {
			if xmlValueMatchRegex.MatchString(value) {
				element.CreateAttr(
					xmlValueMatchRegex.ReplaceAllString(value, "$1"),
					xmlValueMatchRegex.ReplaceAllString(value, "$2"),
				)
			} else {
				element.SetText(value)
			}
		}
	}
	doc.Indent(2)
	return doc.WriteToBytes()
}

// Root.Property='[value="testing"]' sets an XML attribute instead of the text.
var xmlValueMatchRegex = regexp.MustCompile(`^\[([\w]+)='(.*)'\]$`)
