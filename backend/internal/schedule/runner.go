package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/console"
	"app/internal/files"
	"app/internal/serverctl"
	"app/internal/tenancy"
)

// Runner checks all schedules at the start of every minute and runs the due ones.
// It is a manager runnable; runs that were missed while the panel was down are skipped.
type Runner struct {
	Client client.Client
	Reader client.Reader
	Ops    *serverctl.Ops
	// Files creates backups (task action "backup").
	Files    *files.Service
	Hub      *console.Hub
	Trigger  func(namespace, server string)
	Location *time.Location
	Log      *slog.Logger

	mu      sync.Mutex
	running map[string]bool
}

// Start implements manager.Runnable.
func (r *Runner) Start(ctx context.Context) error {
	for {
		next := time.Now().Truncate(time.Minute).Add(time.Minute)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Until(next)):
		}
		r.tick(ctx, next.In(r.Location))
	}
}

func (r *Runner) tick(ctx context.Context, minute time.Time) {
	var list v1alpha1.GameServerList
	if err := r.Client.List(ctx, &list); err != nil {
		r.Log.Warn("listing servers for schedules", "err", err)
		return
	}
	for _, gs := range list.Items {
		if !tenancy.Owns(gs.Namespace) || !gs.DeletionTimestamp.IsZero() || gs.Spec.Suspended {
			continue
		}
		for _, s := range gs.Spec.Schedules {
			if s.Enabled && s.Event == "" && Due(s.Cron, minute) {
				r.start(client.ObjectKeyFromObject(&gs), s)
			}
		}
	}
}

// Errors of RunNow.
var (
	ErrNotFound = errors.New("schedule not found")
	ErrRunning  = errors.New("schedule is already running")
)

// RunNow starts a schedule by hand (also when it is disabled).
func (r *Runner) RunNow(ctx context.Context, gs *v1alpha1.GameServer, name string) error {
	for _, s := range gs.Spec.Schedules {
		if s.Name == name {
			if !r.start(client.ObjectKeyFromObject(gs), s) {
				return fmt.Errorf("%w: %q", ErrRunning, name)
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrNotFound, name)
}

// Started runs the enabled "started" schedules of a server; the controller calls it when the
// server was marked as running.
func (r *Runner) Started(gs *v1alpha1.GameServer) {
	for _, s := range eventSchedules(gs, EventStarted) {
		r.start(client.ObjectKeyFromObject(gs), s)
	}
}

// StartStopping runs the enabled "stopping" schedules of a server and returns how long the
// stop may wait for them at most (0: there are none).
func (r *Runner) StartStopping(gs *v1alpha1.GameServer) time.Duration {
	var wait time.Duration
	for _, s := range eventSchedules(gs, EventStopping) {
		d := 15 * time.Second
		for _, t := range s.Tasks {
			d += time.Duration(t.DelaySeconds)*time.Second + 10*time.Second
		}
		wait = max(wait, d)
		r.start(client.ObjectKeyFromObject(gs), s)
	}
	return wait
}

// Stopping reports whether "stopping" schedules of the server are still running.
func (r *Runner) Stopping(gs *v1alpha1.GameServer) bool {
	for _, s := range eventSchedules(gs, EventStopping) {
		if r.Running(gs, s.Name) {
			return true
		}
	}
	return false
}

func eventSchedules(gs *v1alpha1.GameServer, event string) []v1alpha1.Schedule {
	var out []v1alpha1.Schedule
	for _, s := range gs.Spec.Schedules {
		if s.Enabled && s.Event == event {
			out = append(out, s)
		}
	}
	return out
}

// Running reports whether a schedule is in progress.
func (r *Runner) Running(gs *v1alpha1.GameServer, name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running[key(gs.Namespace, gs.Name, name)]
}

func key(ns, server, schedule string) string { return ns + "/" + server + "/" + schedule }

func (r *Runner) lock(k string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running == nil {
		r.running = map[string]bool{}
	}
	if r.running[k] {
		return false
	}
	r.running[k] = true
	return true
}

func (r *Runner) unlock(k string) {
	r.mu.Lock()
	delete(r.running, k)
	r.mu.Unlock()
}

// start marks the schedule as running and runs it in the background; false when the previous
// run is still in progress. Marking before the goroutine starts lets Running see it right away.
func (r *Runner) start(ref types.NamespacedName, s v1alpha1.Schedule) bool {
	k := key(ref.Namespace, ref.Name, s.Name)
	if !r.lock(k) {
		return false
	}
	go func() {
		defer r.unlock(k)
		r.run(context.Background(), ref, s)
	}()
	return true
}

// run executes the tasks of a schedule one after another; a failing task ends the run.
func (r *Runner) run(ctx context.Context, ref types.NamespacedName, s v1alpha1.Schedule) {
	gs := &v1alpha1.GameServer{}
	if err := r.Reader.Get(ctx, ref, gs); err != nil {
		return
	}
	if s.OnlyWhenOnline && gs.Status.Phase != v1alpha1.PhaseRunning {
		r.record(ctx, ref, s.Name, "skipped: server offline")
		return
	}
	r.Hub.Daemon(ref.Name, fmt.Sprintf("Running schedule %q...", s.Name))
	for i, t := range s.Tasks {
		if t.DelaySeconds > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(t.DelaySeconds) * time.Second):
			}
		}
		if err := r.Reader.Get(ctx, ref, gs); err != nil {
			return // server deleted meanwhile
		}
		var err error
		switch t.Action {
		case "command":
			err = r.Ops.Command(ctx, gs, t.Payload)
		case "backup":
			err = r.backup(ctx, gs, t.Payload)
		default:
			err = r.Ops.Power(ctx, gs, t.Action)
			r.Trigger(ref.Namespace, ref.Name)
		}
		if err != nil {
			result := fmt.Sprintf("failed: task %d (%s): %v", i+1, t.Action, err)
			r.Hub.Daemon(ref.Name, fmt.Sprintf("Schedule %q %s", s.Name, result))
			r.record(ctx, ref, s.Name, result)
			return
		}
	}
	r.record(ctx, ref, s.Name, "ok")
}

// record stores the result of a run in the server status.
func (r *Runner) record(ctx context.Context, ref types.NamespacedName, name, result string) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		gs := &v1alpha1.GameServer{}
		if err := r.Reader.Get(ctx, ref, gs); err != nil {
			return err
		}
		entry := v1alpha1.ScheduleStatus{Name: name, LastRunAt: &metav1.Time{Time: time.Now()}, LastResult: result}
		found := false
		for i := range gs.Status.Schedules {
			if gs.Status.Schedules[i].Name == name {
				gs.Status.Schedules[i], found = entry, true
			}
		}
		if !found {
			gs.Status.Schedules = append(gs.Status.Schedules, entry)
		}
		return r.Client.Status().Update(ctx, gs)
	})
	if err != nil {
		r.Log.Warn("storing schedule result", "server", ref.Name, "schedule", name, "err", err)
	}
}

// backup creates a backup and waits for it (like a backup started on the Backups tab).
func (r *Runner) backup(ctx context.Context, gs *v1alpha1.GameServer, label string) error {
	if r.Files == nil {
		return errors.New("backups are not available")
	}
	return r.Files.RunBackup(ctx, files.RefOf(gs), label, time.Now().In(r.Location))
}
