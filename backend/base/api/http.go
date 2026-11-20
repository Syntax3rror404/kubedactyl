package api

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"slices"
	"strings"
)

// Middleware runs in front of a handler: it answers the request itself (a.Fail) or passes it on to next.
type Middleware func(next http.Handler) http.Handler

// Mux registers the API routes on an http.ServeMux below a path prefix (/api). Every route runs through the
// middleware of the group it was registered in (With).
type Mux struct {
	mux    *http.ServeMux
	prefix string
	chain  []Middleware
	// patterns are the routes registered in all groups.
	patterns *[]string
}

// NewMux returns the root group: routes registered on it are served by mux below prefix.
func NewMux(mux *http.ServeMux, prefix string) *Mux {
	return &Mux{mux: mux, prefix: prefix, patterns: new([]string)}
}

// With returns a group whose routes run through the middleware of m and then mw, in this order.
func (m *Mux) With(mw ...Middleware) *Mux {
	return &Mux{mux: m.mux, prefix: m.prefix, chain: append(slices.Clip(m.chain), mw...), patterns: m.patterns}
}

// Patterns are the registered routes ("GET /api/servers/{server}"), so a test can check every one.
func (m *Mux) Patterns() []string { return slices.Clone(*m.patterns) }

// Handle registers h for a pattern "METHOD /path" with the path below the prefix ("GET /servers/{server}").
func (m *Mux) Handle(pattern string, h http.HandlerFunc) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		panic("api: route without method: " + pattern)
	}
	pattern = method + " " + m.prefix + path
	m.mux.Handle(pattern, Chain(h, m.chain...))
	*m.patterns = append(*m.patterns, pattern)
}

// Chain wraps h in the middleware, the first one outermost: it runs first and sees the answer last.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for _, m := range slices.Backward(mw) {
		h = m(h)
	}
	return h
}

// Guard is middleware that passes a request on only when check finds nothing wrong; otherwise the error is the
// answer.
func (a *API) Guard(check func(r *http.Request) error) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := check(r); err != nil {
				a.Fail(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// jsonOut are the rules of every JSON answer. Empty lists and maps are [] and {}, never null. Like before
// encoding/json/v2: omitempty leaves out false and 0, a duration is a number of nanoseconds and invalid UTF-8 (file
// names on a volume need not be UTF-8) becomes U+FFFD. Map keys are sorted, so equal answers have equal ETags.
var jsonOut = json.JoinOptions(
	jsonv1.OmitEmptyWithLegacySemantics(true),
	jsonv1.FormatDurationAsNano(true),
	jsontext.AllowInvalidUTF8(true),
	json.Deterministic(true),
)

// marshal encodes an answer with jsonOut.
func marshal(v any) ([]byte, error) { return json.Marshal(v, jsonOut) }

// JSON answers with status and v as JSON.
func JSON(w http.ResponseWriter, status int, v any) {
	body, err := marshal(v)
	if err != nil {
		// Only a type that cannot be encoded fails here (a bug): the recovery middleware answers 500 and logs it.
		panic(err)
	}
	Data(w, status, "application/json; charset=utf-8", body)
}

// Data answers with status and body of the content type.
func Data(w http.ResponseWriter, status int, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	// A failed write means the client is gone; there is nobody left to answer.
	_, _ = w.Write(body)
}

// Download sends what write produces as the file name. A failure before the first byte is answered with a.Fail
// (an error, not a broken file the browser would save); a later one only ends the download. Nil: the file was sent.
func (a *API) Download(w http.ResponseWriter, r *http.Request, name string, write func(out io.Writer) error) error {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	out := &sentWriter{Writer: w}
	err := write(out)
	if err != nil && !out.sent {
		w.Header().Del("Content-Disposition")
		w.Header().Del("Content-Type")
		a.Fail(w, r, err)
	}
	return err
}

// Copy is the write function of a Download that sends src.
func Copy(src io.Reader) func(out io.Writer) error {
	return func(out io.Writer) error {
		_, err := io.Copy(out, src)
		return err
	}
}

// unknownField names a member the request does not have by its path ("unknown field \"tasks.0.comand\""),
// not by the Go types; other errors stay as they are.
func unknownField(err error) error {
	var se *json.SemanticError
	if !errors.As(err, &se) || !errors.Is(se.Err, json.ErrUnknownName) {
		return err
	}
	return fmt.Errorf("unknown field %q", strings.ReplaceAll(strings.TrimPrefix(string(se.JSONPointer), "/"), "/", "."))
}

// Uploads calls write for every file of the multipart form field, one after the other as they arrive (nothing
// is buffered). name is the file's base name as the browser sent it: write checks it.
func Uploads(r *http.Request, field string, write func(name string, body io.Reader) error) error {
	reader, err := r.MultipartReader()
	if err != nil {
		return BadRequest(err)
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return BadRequest(err)
		}
		if part.FormName() == field {
			err = write(path.Base(strings.ReplaceAll(part.FileName(), `\`, "/")), part)
		}
		_ = part.Close()
		if err != nil {
			return err
		}
	}
}

// sentWriter tells whether anything was written.
type sentWriter struct {
	io.Writer
	sent bool
}

func (s *sentWriter) Write(p []byte) (int, error) {
	s.sent = true
	return s.Writer.Write(p)
}

// Validator is a request body with rules beyond its JSON types (required fields, lengths, fixed values).
type Validator interface {
	Validate() error
}

// Decode reads the JSON body into req and checks it with its Validate method; a body that does not fit is
// answered with 400 (with the messages per field).
func (a *API) Decode(w http.ResponseWriter, r *http.Request, req any) bool {
	return a.decode(w, r, req, false)
}

// DecodeOptional is Decode for a body that may be left out: without one, req keeps its zero values.
func (a *API) DecodeOptional(w http.ResponseWriter, r *http.Request, req any) bool {
	return a.decode(w, r, req, true)
}

// decode reads a request body strictly: unknown or repeated members, invalid UTF-8 and anything after the value
// are errors, so a misspelled field is reported instead of silently left out.
func (a *API) decode(w http.ResponseWriter, r *http.Request, req any, optional bool) bool {
	body, err := io.ReadAll(r.Body)
	if err == nil && (!optional || len(bytes.TrimSpace(body)) > 0) {
		err = json.Unmarshal(body, req, json.RejectUnknownMembers(true))
	}
	if v, ok := req.(Validator); ok && err == nil {
		err = v.Validate()
	}
	if err != nil {
		a.Fail(w, r, BadRequest(unknownField(err)))
		return false
	}
	return true
}
