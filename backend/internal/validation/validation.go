// Package validation holds the error type of input validation: one message per field, so the
// web interface can show each message next to its input. The API answers these errors with 422
// and the messages in "fields".
package validation

import (
	"fmt"
	"slices"
)

// Error is implemented by every validation error with field messages (Errors, egg.FieldErrors).
type Error interface {
	error
	Fields() map[string]string
}

// Errors maps a field name of the request ("username", "storageClasses") to a message.
type Errors map[string]string

// Field is the validation error of a single field.
func Field(field string, err error) Errors { return Errors{field: err.Error()} }

// Fields returns the messages per field.
func (e Errors) Fields() map[string]string { return e }

// Error names the first field (sorted) and how many more there are.
func (e Errors) Error() string { return Summary(e, func(field string) string { return field }) }

// OrNil returns nil when no field has an error, so a validator can end with `return errs.OrNil()`.
func (e Errors) OrNil() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

// Summary is the one-line form of field messages, e.g. "Author: must be an e-mail address (and 3 more)";
// label turns a field name into what the user sees.
func Summary(fields map[string]string, label func(string) string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if len(keys) == 0 {
		return "invalid input"
	}
	msg := fmt.Sprintf("%s: %s", label(keys[0]), fields[keys[0]])
	if len(keys) > 1 {
		msg += fmt.Sprintf(" (and %d more)", len(keys)-1)
	}
	return msg
}
