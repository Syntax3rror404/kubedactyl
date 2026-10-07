package files

import (
	"context"
	"crypto/rand"
	"errors"
	"slices"
	"strings"
	"time"
)

// ErrJobNotFound is returned for a job that is not (or no longer) in the list of a server.
var ErrJobNotFound = errors.New("job not found")

// markerPrefix starts the name ($0) the scripts of a job run under, so Cancel finds their processes.
const markerPrefix = "kd-job-"

// cancelGrace is how long a cancelled job may clean up before its stream is closed.
const cancelGrace = 5 * time.Second

// cancelScript sends SIGTERM to the process group of every script named "kd-job-$1". Each script
// started through exec leads its own group, which also holds its pipes and background processes
// (tar, gzip, wget). The token comes without its prefix, so this script never matches itself.
const cancelScript = `for p in /proc/[0-9]*; do
  { tr '\0' '\n' <"$p/cmdline"; } 2>/dev/null | grep -qxF "kd-job-$1" && kill -TERM -"${p#/proc/}" 2>/dev/null
done
exit 0`

type markerKey struct{}

func withMarker(ctx context.Context, marker string) context.Context {
	return context.WithValue(ctx, markerKey{}, marker)
}

// scriptName is the $0 of the scripts run with ctx: the marker of its job, or "sh".
func scriptName(ctx context.Context) string {
	if m, ok := ctx.Value(markerKey{}).(string); ok {
		return m
	}
	return "sh"
}

// track remembers how to stop a job that starts and returns the name its scripts run under. A job
// cancelled before it started stops at once.
func (s *Service) track(j *Job, cancel context.CancelFunc) string {
	marker := markerPrefix + rand.Text()
	s.jobs.mu.Lock()
	j.cancel, j.marker = cancel, marker
	cancelled := j.cancelled
	s.jobs.mu.Unlock()
	if cancelled {
		cancel()
	}
	return marker
}

// Cancel stops a running job of the server: its processes in the files pod get SIGTERM, the scripts
// remove what they leave half done (a restore leaves the server files incomplete) and the job ends
// as cancelled. Cancelling a finished job does nothing.
func (s *Service) Cancel(ctx context.Context, ref Ref, id string) error {
	s.jobs.mu.Lock()
	i := slices.IndexFunc(s.jobs.jobs[ref], func(j *Job) bool { return j.ID == id })
	if i < 0 {
		s.jobs.mu.Unlock()
		return ErrJobNotFound
	}
	j := s.jobs.jobs[ref][i]
	if j.State != JobRunning {
		s.jobs.mu.Unlock()
		return nil
	}
	j.cancelled = true
	cancel, marker := j.cancel, j.marker
	s.jobs.mu.Unlock()
	if cancel == nil {
		return nil // track stops it
	}
	// Without the pod there is nothing to stop; closing the stream ends the job in any case.
	_ = s.Manager.run(ctx, ref, nil, nil, cancelScript, strings.TrimPrefix(marker, markerPrefix))
	time.AfterFunc(cancelGrace, cancel)
	return nil
}
