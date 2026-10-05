package egg

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// SplitRules splits Laravel rules on "|" while keeping "|" inside regex:/.../ intact.
func SplitRules(rules string) []string {
	var out []string
	parts := strings.Split(rules, "|")
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if strings.HasPrefix(p, "regex:") || strings.HasPrefix(p, "not_regex:") {
			pattern := p[strings.Index(p, ":")+1:]
			for i+1 < len(parts) && !isCompleteRegex(pattern) {
				i++
				p += "|" + parts[i]
				pattern += "|" + parts[i]
			}
		}
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

var regexFlagsRe = regexp.MustCompile(`^[a-zA-Z]*$`)

// isCompleteRegex reports whether a PHP style pattern /.../flags is closed.
func isCompleteRegex(p string) bool {
	if len(p) < 2 {
		return false
	}
	delim := p[0]
	end := strings.LastIndexByte(p[1:], delim)
	if end < 0 {
		return false
	}
	return regexFlagsRe.MatchString(p[end+2:])
}

// phpRegex converts a PHP pattern like /^abc$/i into a Go regexp.
func phpRegex(p string) (*regexp.Regexp, error) {
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid regex %q", p)
	}
	delim := p[0]
	end := strings.LastIndexByte(p[1:], delim)
	if end < 0 {
		return nil, fmt.Errorf("invalid regex %q", p)
	}
	body, flags := p[1:end+1], p[end+2:]
	prefix := ""
	for _, f := range flags {
		switch f {
		case 'i', 'm', 's':
			prefix += string(f)
		}
	}
	if prefix != "" {
		body = "(?" + prefix + ")" + body
	}
	return regexp.Compile(body)
}

// Validate checks a variable value against the Laravel style rules of an egg variable
// (e.g. "required|string|max:20"). Rules without a check here (string, nullable, …) are ignored.
func Validate(value, rules string) error {
	list := SplitRules(rules)
	if value == "" {
		if hasRule(list, "required") {
			return errors.New("is required")
		}
		return nil
	}
	in := newRuleInput(value, list)
	for _, rule := range list {
		name, arg, _ := strings.Cut(rule, ":")
		check, ok := ruleChecks[name]
		if !ok {
			continue
		}
		in.arg, in.args = arg, strings.Split(arg, ",")
		if err := check(in); err != nil {
			return err
		}
	}
	return nil
}

func hasRule(list []string, name string) bool {
	for _, r := range list {
		if n, _, _ := strings.Cut(r, ":"); n == name {
			return true
		}
	}
	return false
}
