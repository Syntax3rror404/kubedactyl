package api

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestDownload: a download that fails before its first byte is answered with an error, not a broken file; one
// that fails later just ends.
func TestDownload(t *testing.T) {
	a := &API{Log: slog.New(slog.DiscardHandler)}
	download := func(write func(io.Writer) error) (*httptest.ResponseRecorder, error) {
		w := httptest.NewRecorder()
		return w, a.Download(w, httptest.NewRequest(http.MethodGet, "/", nil), "world.zip", write)
	}
	w, err := download(func(out io.Writer) error { return NotFound(errors.New("file not found")) })
	if err == nil || w.Code != http.StatusNotFound || w.Header().Get("Content-Disposition") != "" ||
		!strings.Contains(w.Body.String(), "file not found") {
		t.Errorf("failure before the first byte: %d %v %q", w.Code, w.Header(), w.Body.String())
	}
	w, err = download(func(out io.Writer) error {
		_, _ = out.Write([]byte("PK"))
		return errors.New("connection lost")
	})
	if err == nil || w.Code != http.StatusOK || w.Body.String() != "PK" {
		t.Errorf("failure after the first byte: %d %q", w.Code, w.Body.String())
	}
	w, err = download(Copy(strings.NewReader("data")))
	if err != nil || w.Body.String() != "data" ||
		w.Header().Get("Content-Disposition") != `attachment; filename=world.zip` {
		t.Errorf("download: %v %v", err, w.Header())
	}
}

// TestMuxGroups: a group runs its parent's middleware first, and every route of every group is listed.
func TestMuxGroups(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	mux := http.NewServeMux()
	root := NewMux(mux, "/api")
	signed := root.With(mark("outer"))
	signed.With(mark("inner")).Handle("GET /things/{thing}", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler "+r.PathValue("thing"))
	})
	root.Handle("POST /public", func(http.ResponseWriter, *http.Request) {})
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/things/x", nil))
	if got := strings.Join(order, ", "); got != "outer, inner, handler x" {
		t.Errorf("order: %s", got)
	}
	if got := strings.Join(root.Patterns(), ", "); got != "GET /api/things/{thing}, POST /api/public" {
		t.Errorf("patterns: %s", got)
	}
}

type labelRequest struct {
	Label string `json:"label"`
}

func (req labelRequest) Validate() error {
	if req.Label == "bad" {
		return errors.New("bad label")
	}
	return nil
}

// TestDecodeOptional: a missing body keeps the zero values, a present one is read and checked like with Decode.
func TestDecodeOptional(t *testing.T) {
	a := &API{Log: slog.New(slog.DiscardHandler)}
	for body, want := range map[string]int{"": http.StatusOK, " \n": http.StatusOK, `{"label":"x"}`: http.StatusOK,
		`{"label":"bad"}`: http.StatusBadRequest, `{"label":`: http.StatusBadRequest} {
		var req labelRequest
		w := httptest.NewRecorder()
		ok := a.DecodeOptional(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)), &req)
		if ok != (want == http.StatusOK) || w.Code != want {
			t.Errorf("%q: %v %d, want %d", body, ok, w.Code, want)
		}
	}
	var req labelRequest
	w := httptest.NewRecorder()
	if a.Decode(w, httptest.NewRequest(http.MethodPost, "/", http.NoBody), &req) || w.Code != http.StatusBadRequest {
		t.Errorf("Decode accepted a missing body: %d", w.Code)
	}
}

// TestDecodeIsStrict: a body with a member the request does not have (also in other case), a repeated member,
// invalid UTF-8 or data after the value is refused, naming the problem.
func TestDecodeIsStrict(t *testing.T) {
	a := &API{Log: slog.New(slog.DiscardHandler)}
	for body, want := range map[string]string{
		`{"label":"x","lable":"y"}`: `unknown field \"lable\"`, `{"Label":"x"}`: `unknown field \"Label\"`,
		`{"label":"x","label":"y"}`: "duplicate",
		"{\"label\":\"\xff\"}":      "UTF-8", `{"label":"x"} {}`: "after top-level value",
	} {
		var req labelRequest
		w := httptest.NewRecorder()
		ok := a.Decode(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)), &req)
		if ok || w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), want) {
			t.Errorf("%q: %v %d %s", body, ok, w.Code, w.Body.String())
		}
	}
}

// TestJSONAnswers: empty lists and maps are [] and {}, omitempty leaves out false and 0, durations are
// nanoseconds, invalid UTF-8 is replaced and map keys are sorted.
func TestJSONAnswers(t *testing.T) {
	type answer struct {
		List    []string          `json:"list"`
		Map     map[string]string `json:"map"`
		Count   int               `json:"count,omitempty"`
		On      bool              `json:"on,omitempty"`
		Timeout time.Duration     `json:"timeout"`
		Name    string            `json:"name"`
		Labels  map[string]int    `json:"labels"`
	}
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, answer{Timeout: time.Second, Name: "a\xffb", Labels: map[string]int{"b": 2, "a": 1, "c": 3}})
	want := `{"list":[],"map":{},"timeout":1000000000,"name":"a` + "\uFFFD" + `b","labels":{"a":1,"b":2,"c":3}}`
	if got := w.Body.String(); got != want {
		t.Errorf("answer:\n got %s\nwant %s", got, want)
	}
}

// TestUploads: every file of the field is written in order with its base name; other fields are skipped and an
// error of write ends the upload.
func TestUploads(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, f := range [][3]string{{"files", `C:\dir\a.txt`, "A"}, {"other", "x.txt", "X"}, {"files", "../b.txt", "B"}} {
		part, _ := form.CreateFormFile(f[0], f[1])
		_, _ = part.Write([]byte(f[2]))
	}
	_ = form.Close()
	upload := func(write func(string, io.Reader) error) error {
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
		r.Header.Set("Content-Type", form.FormDataContentType())
		return Uploads(r, "files", write)
	}
	var got []string
	err := upload(func(name string, b io.Reader) error {
		data, err := io.ReadAll(b)
		got = append(got, name+"="+string(data))
		return err
	})
	if err != nil || strings.Join(got, ", ") != "a.txt=A, b.txt=B" {
		t.Errorf("uploads: %v %v", err, got)
	}
	stop := errors.New("bad name")
	if err := upload(func(string, io.Reader) error { return stop }); err != stop {
		t.Errorf("write error: %v", err)
	}
	if err := Uploads(httptest.NewRequest(http.MethodPost, "/", nil), "files", nil); err == nil {
		t.Error("a request without a form passed")
	}
}
