package egg

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ruleInput is what a single rule check needs: the value and the rule's argument.
type ruleInput struct {
	value string
	arg   string   // text after "name:", e.g. "20" or "1,100"
	args  []string // arg split at ","
	// numeric is set when the rules contain numeric/integer: min/max/between then compare the
	// number itself instead of the length (Laravel semantics).
	numeric bool
	number  float64
	isNum   bool
}

func newRuleInput(value string, rules []string) ruleInput {
	n, err := strconv.ParseFloat(value, 64)
	numeric := hasRule(rules, "numeric") || hasRule(rules, "integer") || hasRule(rules, "int")
	return ruleInput{value: value, numeric: numeric, number: n, isNum: err == nil}
}

// size is the number (numeric rules) or the length of the value.
func (in ruleInput) size() float64 {
	if in.numeric {
		return in.number
	}
	return float64(utf8.RuneCountInString(in.value))
}

// unit completes messages like "may not be greater than 20 characters".
func (in ruleInput) unit() string {
	if in.numeric {
		return ""
	}
	return " characters"
}

type ruleCheck func(in ruleInput) error

// ruleChecks maps a rule name to its check. Add new rules here.
var ruleChecks = map[string]ruleCheck{
	"numeric":        checkNumeric,
	"integer":        checkInteger,
	"int":            checkInteger,
	"boolean":        checkBoolean,
	"bool":           checkBoolean,
	"max":            sizeRule(func(size, limit float64) bool { return size <= limit }, "may not be greater than %s%s"),
	"min":            sizeRule(func(size, limit float64) bool { return size >= limit }, "must be at least %s%s"),
	"gt":             sizeRule(func(size, limit float64) bool { return size > limit }, "must be greater than %s%s"),
	"size":           sizeRule(func(size, limit float64) bool { return size == limit }, "must be %s%s"),
	"between":        checkBetween,
	"digits_between": checkDigitsBetween,
	"in":             checkIn,
	"not_in":         checkNotIn,
	"regex":          regexRule(true),
	"not_regex":      regexRule(false),
	"alpha_dash": patternRule(
		regexp.MustCompile(`^[\pL\pM\pN_-]+$`),
		"may only contain letters, numbers, dashes and underscores",
	),
	"alpha_num": patternRule(regexp.MustCompile(`^[\pL\pM\pN]+$`), "may only contain letters and numbers"),
	"url":       checkURL,
	"ends_with": checkEndsWith,
}

func checkNumeric(in ruleInput) error {
	if !in.isNum {
		return errors.New("must be a number")
	}
	return nil
}

func checkInteger(in ruleInput) error {
	if _, err := strconv.ParseInt(in.value, 10, 64); err != nil {
		return errors.New("must be an integer")
	}
	return nil
}

func checkBoolean(in ruleInput) error {
	switch strings.ToLower(in.value) {
	case "0", "1", "true", "false":
		return nil
	}
	return errors.New("must be true/false or 1/0")
}

// sizeRule builds min/max/gt/size: ok compares the size of the value with the rule's limit.
// A limit that is no number makes the rule a no-op (like before).
func sizeRule(ok func(size, limit float64) bool, message string) ruleCheck {
	return func(in ruleInput) error {
		limit, err := strconv.ParseFloat(in.arg, 64)
		if err == nil && !ok(in.size(), limit) {
			return fmt.Errorf(message, in.arg, in.unit())
		}
		return nil
	}
}

func checkBetween(in ruleInput) error {
	if len(in.args) != 2 {
		return nil
	}
	lo, e1 := strconv.ParseFloat(in.args[0], 64)
	hi, e2 := strconv.ParseFloat(in.args[1], 64)
	if e1 == nil && e2 == nil && (in.size() < lo || in.size() > hi) {
		return fmt.Errorf("must be between %s and %s%s", in.args[0], in.args[1], in.unit())
	}
	return nil
}

var digitsRe = regexp.MustCompile(`^\d+$`)

func checkDigitsBetween(in ruleInput) error {
	if len(in.args) != 2 {
		return nil
	}
	lo, _ := strconv.Atoi(in.args[0])
	hi, _ := strconv.Atoi(in.args[1])
	if !digitsRe.MatchString(in.value) || len(in.value) < lo || len(in.value) > hi {
		return fmt.Errorf("must have between %s and %s digits", in.args[0], in.args[1])
	}
	return nil
}

func checkIn(in ruleInput) error {
	if !slices.Contains(in.args, in.value) {
		return fmt.Errorf("must be one of: %s", strings.Join(in.args, ", "))
	}
	return nil
}

func checkNotIn(in ruleInput) error {
	if slices.Contains(in.args, in.value) {
		return fmt.Errorf("may not be one of: %s", strings.Join(in.args, ", "))
	}
	return nil
}

// regexRule builds regex (must match) and not_regex (must not match); an invalid pattern is ignored.
func regexRule(mustMatch bool) ruleCheck {
	return func(in ruleInput) error {
		re, err := phpRegex(in.arg)
		if err == nil && re.MatchString(in.value) != mustMatch {
			return errors.New("has an invalid format")
		}
		return nil
	}
}

func patternRule(re *regexp.Regexp, message string) ruleCheck {
	return func(in ruleInput) error {
		if !re.MatchString(in.value) {
			return fmt.Errorf("%s", message)
		}
		return nil
	}
}

func checkURL(in ruleInput) error {
	if u, err := url.ParseRequestURI(in.value); err != nil || u.Host == "" {
		return errors.New("must be a valid URL")
	}
	return nil
}

func checkEndsWith(in ruleInput) error {
	for _, suffix := range in.args {
		if strings.HasSuffix(in.value, suffix) {
			return nil
		}
	}
	return fmt.Errorf("must end with one of: %s", strings.Join(in.args, ", "))
}
