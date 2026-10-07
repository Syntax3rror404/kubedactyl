package files

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

// Progress units.
const (
	ProgressFiles = "files"
	ProgressBytes = "bytes"
)

// Progress tells how far a job is: Done of Total in Unit (files packed, or bytes of an archive read
// or a download written; Total 0 when the size is unknown), File is the entry tar works on.
// StartedAt is the first report (a backup counts its files before), so speed and time left can be
// measured from there.
type Progress struct {
	Unit      string    `json:"unit"           enums:"files,bytes"`
	Done      int64     `json:"done"`
	Total     int64     `json:"total"`
	File      string    `json:"file,omitempty"`
	StartedAt time.Time `json:"startedAt"`
}

// maxProgressLine drops a line that never ends (a path is at most a few KiB).
const maxProgressLine = 64 << 10

// progressWriter reads the output of the backup, restore and download scripts line by line:
// "total N" (files to pack), "pos N SIZE" (bytes read or written, SIZE 0 if unknown) and
// otherwise the entry tar works on (one line per entry, folders end with "/"). Every change goes
// to report.
type progressWriter struct {
	report func(Progress)
	p      Progress
	line   []byte
}

func (w *progressWriter) Write(b []byte) (int, error) {
	w.line = append(w.line, b...)
	for {
		i := bytes.IndexByte(w.line, '\n')
		if i < 0 {
			break
		}
		w.parse(string(w.line[:i]))
		w.line = append(w.line[:0], w.line[i+1:]...)
	}
	if len(w.line) > maxProgressLine {
		w.line = w.line[:0]
	}
	return len(b), nil
}

func (w *progressWriter) parse(line string) {
	f := strings.Fields(line)
	switch {
	case len(f) == 2 && f[0] == "total":
		total, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			return
		}
		w.p.Unit, w.p.Total = ProgressFiles, total
	case len(f) > 0 && f[0] == "pos":
		// The read position can be missing for a moment (the restore just started or ended).
		if len(f) != 3 {
			return
		}
		done, err1 := strconv.ParseInt(f[1], 10, 64)
		total, err2 := strconv.ParseInt(f[2], 10, 64)
		if err1 != nil || err2 != nil {
			return
		}
		w.p.Unit, w.p.Done, w.p.Total = ProgressBytes, done, total
	case line == "" || line == "./":
		return
	default:
		w.p.File = strings.TrimPrefix(line, "./")
		if w.p.Unit == ProgressFiles && !strings.HasSuffix(line, "/") {
			w.p.Done++
		}
	}
	w.report(w.p)
}
