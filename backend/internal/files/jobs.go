package files

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Job kinds.
const (
	JobPull    = "pull"
	JobBackup  = "backup"
	JobRestore = "restore"
)

// Job states.
const (
	JobRunning = "running"
	JobDone    = "done"
	JobFailed  = "failed"
)

// Job is a long file operation of a server that runs in the background.
type Job struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"  enums:"pull,backup,restore"`
	Label string `json:"label"`
	// State is running, done or failed.
	State      string     `json:"state"                enums:"running,done,failed"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// Errors of jobs.
var (
	ErrBusy      = errors.New("a backup or restore of this server is already running")
	ErrRestoring = errors.New("a backup is being restored, wait until it has finished")
)

const (
	keptJobs = 20
	jobLimit = time.Hour
)

type jobStore struct {
	mu   sync.Mutex
	jobs map[Ref][]*Job
	seq  atomic.Int64
}

// Jobs returns the recent jobs of a server, newest first.
func (s *Service) Jobs(ref Ref) []Job {
	s.jobs.mu.Lock()
	defer s.jobs.mu.Unlock()
	out := []Job{}
	list := s.jobs.jobs[ref]
	for i := len(list) - 1; i >= 0; i-- {
		out = append(out, *list[i])
	}
	return out
}

// Busy reports whether a job of one of the kinds is running for the server.
func (s *Service) Busy(ref Ref, kinds ...string) bool {
	s.jobs.mu.Lock()
	defer s.jobs.mu.Unlock()
	return s.busyLocked(ref, kinds...)
}

func (s *Service) busyLocked(ref Ref, kinds ...string) bool {
	for _, j := range s.jobs.jobs[ref] {
		if j.State == JobRunning && slices.Contains(kinds, j.Kind) {
			return true
		}
	}
	return false
}

// Start runs fn in the background. Backups and restores exclude each other per server. The
// files pod is started first and kept alive while the job runs.
func (s *Service) Start(ref Ref, kind, label string, fn func(ctx context.Context) error) (Job, error) {
	j, err := s.add(ref, kind, label)
	if err != nil {
		return Job{}, err
	}
	// The job outlives the request that started it; its error is stored in the job.
	go func() { _ = s.run(context.Background(), ref, j, fn) }()
	return *j, nil
}

// Run is Start that waits for the job (used by schedules).
func (s *Service) Run(ctx context.Context, ref Ref, kind, label string, fn func(ctx context.Context) error) error {
	j, err := s.add(ref, kind, label)
	if err != nil {
		return err
	}
	return s.run(ctx, ref, j, fn)
}

// add registers a running job; the busy check and the registration happen under one lock.
func (s *Service) add(ref Ref, kind, label string) (*Job, error) {
	s.jobs.mu.Lock()
	defer s.jobs.mu.Unlock()
	if (kind == JobBackup || kind == JobRestore) && s.busyLocked(ref, JobBackup, JobRestore) {
		return nil, ErrBusy
	}
	j := &Job{
		ID:        strconv.FormatInt(s.jobs.seq.Add(1), 10),
		Kind:      kind,
		Label:     label,
		State:     JobRunning,
		StartedAt: time.Now().UTC(),
	}
	if s.jobs.jobs == nil {
		s.jobs.jobs = map[Ref][]*Job{}
	}
	list := append(s.jobs.jobs[ref], j)
	if len(list) > keptJobs {
		list = list[len(list)-keptJobs:]
	}
	s.jobs.jobs[ref] = list
	return j, nil
}

func (s *Service) run(ctx context.Context, ref Ref, j *Job, fn func(ctx context.Context) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, jobLimit)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job panicked: %v", r)
		}
		s.finish(j, err)
	}()
	if err = s.EnsurePod(ctx, ref); err != nil {
		return err
	}
	// Keep the files pod alive while the job runs.
	done := make(chan struct{})
	defer close(done)
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				s.Activity.Touch(ref)
			}
		}
	}()
	err = fn(ctx)
	s.Activity.Touch(ref)
	return err
}

func (s *Service) finish(j *Job, err error) {
	s.jobs.mu.Lock()
	defer s.jobs.mu.Unlock()
	now := time.Now().UTC()
	j.FinishedAt = &now
	j.State = JobDone
	if err != nil {
		j.State, j.Error = JobFailed, err.Error()
	}
}
