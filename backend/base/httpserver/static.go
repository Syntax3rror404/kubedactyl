package httpserver

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// compressible are the file types of the web UI that gzip makes smaller (fonts and images are compressed already).
var compressible = []string{".html", ".js", ".css", ".svg", ".json", ".md", ".txt"}

// gzipped is a file of the web UI compressed once at start.
type gzipped struct {
	data        []byte
	contentType string
}

// staticFiles serves the embedded web UI. Files under assets/ carry a content hash in their name and are cached
// for a year; the others are revalidated with their ETag (304 while unchanged). Browsers that accept gzip get
// text files compressed.
type staticFiles struct {
	fs      fs.FS
	etags   map[string]string
	gzipped map[string]gzipped
	// index is index.html, read once (nil when the build has none).
	index []byte
}

func newStaticFiles(frontend fs.FS) (*staticFiles, error) {
	s := &staticFiles{fs: frontend, etags: map[string]string{}, gzipped: map[string]gzipped{}}
	err := fs.WalkDir(frontend, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(frontend, name)
		if err != nil {
			return err
		}
		if name == "index.html" {
			s.index = data
		}
		sum := sha256.Sum256(data)
		// Weak: the same file sent compressed or not.
		s.etags[name] = `W/"` + hex.EncodeToString(sum[:16]) + `"`
		if !compressibleFile(name) {
			return nil
		}
		var buf bytes.Buffer
		w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if _, err := w.Write(data); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		if buf.Len() < len(data) {
			s.gzipped[name] = gzipped{data: buf.Bytes(), contentType: contentType(name, data)}
		}
		return nil
	})
	return s, err
}

func compressibleFile(name string) bool {
	for _, ext := range compressible {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// contentType is the type of a file by its extension, else by its content (e.g. Markdown).
func contentType(name string, data []byte) string {
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		return t
	}
	if strings.HasSuffix(name, ".md") {
		return "text/markdown; charset=utf-8"
	}
	return http.DetectContentType(data)
}

// serve answers with a file of the build and falls back to index.html for unknown paths (client-side routing).
// A missing asset is a 404: a tab of an older build asks for chunks that are gone, and index.html would reach it as
// a script with the wrong type.
func (s *staticFiles) serve(c *gin.Context) {
	name := strings.TrimPrefix(c.Request.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	info, err := fs.Stat(s.fs, name)
	if errors.Is(err, fs.ErrNotExist) && strings.HasPrefix(name, "assets/") && path.Ext(name) != "" {
		c.Header("Cache-Control", "no-store")
		c.String(http.StatusNotFound, "Not found")
		return
	}
	// Folders are no files either: no listing of assets/.
	if errors.Is(err, fs.ErrNotExist) || (err == nil && info.IsDir()) {
		name = "index.html"
	} else if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if strings.HasPrefix(name, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// index.html and the files next to it keep their names: always ask whether there is a newer version.
		c.Header("Cache-Control", "no-cache")
	}
	gz, compressed := s.gzipped[name]
	if compressed {
		c.Header("Vary", "Accept-Encoding")
	}
	c.Header("ETag", s.etags[name])
	if match := c.GetHeader("If-None-Match"); match != "" && strings.Contains(match, s.etags[name]) {
		c.Status(http.StatusNotModified)
		return
	}
	if compressed && strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		c.Header("Content-Encoding", "gzip")
		c.Data(http.StatusOK, gz.contentType, gz.data)
		return
	}
	s.serveFile(c, name)
}

// serveFile sends a file as it is. index.html is sent from memory: http.FileServer redirects it to "/".
func (s *staticFiles) serveFile(c *gin.Context, name string) {
	switch {
	case name != "index.html":
		c.FileFromFS(name, http.FS(s.fs))
	case s.index == nil:
		c.String(http.StatusInternalServerError, "index.html is missing. Was the frontend built?")
	default:
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.index)
	}
}
