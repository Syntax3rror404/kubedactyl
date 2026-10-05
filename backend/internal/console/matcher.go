package console

import (
	"regexp"
	"strings"
)

// Matcher detects the configured "done" lines of an egg.
type Matcher struct {
	plain     []string
	regex     []*regexp.Regexp
	stripAnsi bool
}

// NewMatcher builds a matcher from egg config.startup.done entries.
func NewMatcher(done []string, stripAnsi bool) *Matcher {
	m := &Matcher{stripAnsi: stripAnsi}
	for _, d := range done {
		if expr, ok := strings.CutPrefix(d, "regex:"); ok {
			if re, err := regexp.Compile(expr); err == nil {
				m.regex = append(m.regex, re)
			}
			continue
		}
		m.plain = append(m.plain, d)
	}
	return m
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b[()][0-9A-Za-z]|\x1b[=>]`)

// StripAnsi removes terminal escape sequences.
func StripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

// Match reports whether the line marks the server as started.
func (m *Matcher) Match(line string) bool {
	if m == nil {
		return false
	}
	if m.stripAnsi {
		line = StripAnsi(line)
	}
	for _, p := range m.plain {
		if strings.Contains(line, p) {
			return true
		}
	}
	for _, re := range m.regex {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
