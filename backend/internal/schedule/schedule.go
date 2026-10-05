// Package schedule runs the tasks of game server schedules:
// console commands and power actions at times given by cron expressions, or at events of the
// server's life: after it started and before it stops ("Tasks" in the web interface).
package schedule

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"app/api/v1alpha1"
	"app/internal/serverctl"
)

// Limits (the CRD validates the same).
const (
	MaxSchedules = 20
	MaxTasks     = 10
	MaxDelay     = 900
	MaxCommand   = 500
	// MaxStopDelay limits the delays of the "stopping" tasks: the stop waits for them.
	MaxStopDelay = 300
)

// Events a schedule can run at instead of a cron expression.
const (
	// EventStarted: the server was marked as running (its "done" line appeared).
	EventStarted = "started"
	// EventStopping: the server is about to be stopped or restarted; the stop waits for the tasks.
	EventStopping = "stopping"
)

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// Parse accepts five-field cron expressions ("0 4 * * *") and descriptors ("@daily").
// Time zone prefixes are rejected: all schedules use the panel's time zone.
func Parse(expr string) (cron.Schedule, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(strings.ToUpper(expr), "CRON_TZ=") || strings.HasPrefix(strings.ToUpper(expr), "TZ=") {
		return nil, errors.New("time zones are not supported in the expression; the panel time zone is used")
	}
	if strings.HasPrefix(expr, "@every") {
		return nil, errors.New("@every is not supported; use a cron expression")
	}
	return parser.Parse(expr)
}

// Next returns the next run after t (in the location of t), or zero for invalid expressions.
func Next(expr string, t time.Time) time.Time {
	s, err := Parse(expr)
	if err != nil {
		return time.Time{}
	}
	return s.Next(t)
}

// Due reports whether the schedule runs at the given minute.
func Due(expr string, minute time.Time) bool {
	s, err := Parse(expr)
	if err != nil {
		return false
	}
	return s.Next(minute.Add(-time.Second)).Equal(minute)
}

// ErrInvalid marks schedules that fail validation (the message says what is wrong).
var ErrInvalid = errors.New("invalid schedule")

type invalid struct{ error }

func (e invalid) Is(target error) bool { return target == ErrInvalid }
func (e invalid) Unwrap() error        { return e.error }

// Validate checks a server's schedules and normalizes names and expressions.
func Validate(list []v1alpha1.Schedule) ([]v1alpha1.Schedule, error) {
	out, err := validate(list)
	if err != nil {
		return nil, invalid{err}
	}
	return out, nil
}

func validate(list []v1alpha1.Schedule) ([]v1alpha1.Schedule, error) {
	if len(list) > MaxSchedules {
		return nil, fmt.Errorf("at most %d schedules per server", MaxSchedules)
	}
	out := make([]v1alpha1.Schedule, 0, len(list))
	seen := map[string]bool{}
	for i, s := range list {
		s, err := validateSchedule(i, s, seen)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// validateSchedule checks one schedule; seen holds the lower-case names of the previous ones.
func validateSchedule(i int, s v1alpha1.Schedule, seen map[string]bool) (v1alpha1.Schedule, error) {
	s.Name = strings.TrimSpace(s.Name)
	s.Cron = strings.Join(strings.Fields(s.Cron), " ")
	label := fmt.Sprintf("schedule %d", i+1)
	if s.Name != "" {
		label = fmt.Sprintf("schedule %q", s.Name)
	}
	switch {
	case s.Name == "" || len([]rune(s.Name)) > 50:
		return s, fmt.Errorf("%s: the name must have 1 to 50 characters", label)
	case seen[strings.ToLower(s.Name)]:
		return s, fmt.Errorf("%s: the name is used twice", label)
	}
	seen[strings.ToLower(s.Name)] = true
	if err := validateTrigger(&s); err != nil {
		return s, fmt.Errorf("%s: %w", label, err)
	}
	if len(s.Tasks) == 0 || len(s.Tasks) > MaxTasks {
		return s, fmt.Errorf("%s: 1 to %d tasks", label, MaxTasks)
	}
	for j, t := range s.Tasks {
		t, err := validateTask(t)
		if err != nil {
			return s, fmt.Errorf("%s, task %d: %w", label, j+1, err)
		}
		s.Tasks[j] = t
	}
	return s, validateEventTasks(s)
}

// validateTrigger requires either a valid cron expression or a known event.
func validateTrigger(s *v1alpha1.Schedule) error {
	switch {
	case s.Event != "" && s.Cron != "":
		return errors.New("set either a cron expression or an event, not both")
	case s.Event != "":
		if s.Event != EventStarted && s.Event != EventStopping {
			return fmt.Errorf("unknown event %q", s.Event)
		}
		s.OnlyWhenOnline = false
		return nil
	}
	if _, err := Parse(s.Cron); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

// validateEventTasks allows only console commands at events (a power action in a "stopping"
// task would stop the server again) and limits how long a stop may wait.
func validateEventTasks(s v1alpha1.Schedule) error {
	if s.Event == "" {
		return nil
	}
	delay := int32(0)
	for j, t := range s.Tasks {
		if t.Action != "command" {
			return fmt.Errorf("schedule %q, task %d: only console commands can run at events", s.Name, j+1)
		}
		delay += t.DelaySeconds
	}
	if s.Event == EventStopping && delay > MaxStopDelay {
		return fmt.Errorf("schedule %q: the delays of stop tasks may add up to %d seconds", s.Name, MaxStopDelay)
	}
	return nil
}

// validateTask checks a task; only commands and backups keep a payload.
func validateTask(t v1alpha1.ScheduleTask) (v1alpha1.ScheduleTask, error) {
	t.Payload = strings.TrimSpace(t.Payload)
	switch {
	case t.Action == "command" && (t.Payload == "" || len(t.Payload) > MaxCommand):
		return t, fmt.Errorf("the command must have 1 to %d characters", MaxCommand)
	case t.Action == "backup" && len(t.Payload) > 40:
		return t, errors.New("the backup label must have at most 40 characters")
	case t.Action != "command" && t.Action != "backup" && !slices.Contains(serverctl.Signals, t.Action):
		return t, fmt.Errorf("unknown action %q", t.Action)
	case t.DelaySeconds < 0 || t.DelaySeconds > MaxDelay:
		return t, fmt.Errorf("the delay must be 0 to %d seconds", MaxDelay)
	}
	if t.Action != "command" && t.Action != "backup" {
		t.Payload = ""
	}
	return t, nil
}

// Location loads the panel time zone (UTC when unknown).
func Location(name string) *time.Location {
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.UTC
}
