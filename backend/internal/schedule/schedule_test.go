package schedule

import (
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/console"
	"app/internal/serverctl"
	"app/internal/tenancy"
	"app/internal/testutil"
)

func TestDue(t *testing.T) {
	berlin := Location("Europe/Berlin")
	at := func(s string) time.Time { tt, _ := time.ParseInLocation("2006-01-02 15:04", s, berlin); return tt }
	cases := []struct {
		expr string
		at   string
		want bool
	}{
		{"0 4 * * *", "2026-09-29 04:00", true},
		{"0 4 * * *", "2026-09-29 04:01", false},
		{"@daily", "2026-09-29 00:00", true},
		{"@hourly", "2026-09-29 13:00", true},
		{"*/15 * * * *", "2026-09-29 13:45", true},
		{"*/15 * * * *", "2026-09-29 13:46", false},
		{"0 6 * * 1", "2026-09-28 06:00", true}, // Monday
		{"0 6 * * 1", "2026-09-29 06:00", false},
	}
	for _, c := range cases {
		if got := Due(c.expr, at(c.at)); got != c.want {
			t.Errorf("Due(%q, %s) = %v", c.expr, c.at, got)
		}
	}
	if next := Next("30 3 * * *", at("2026-09-29 04:00")); !next.Equal(at("2026-09-30 03:30")) {
		t.Errorf("Next = %s", next)
	}
	if Location("Nowhere/Invalid") != time.UTC {
		t.Error("unknown time zones fall back to UTC")
	}
}

func TestValidate(t *testing.T) {
	ok := v1alpha1.Schedule{
		Name:    " Daily restart ",
		Cron:    " 0  4 * * * ",
		Enabled: true,
		Tasks: []v1alpha1.ScheduleTask{
			{
				Action:  "command",
				Payload: " say Restart in 5 minutes ",
			},
			{Action: "restart", Payload: "ignored", DelaySeconds: 300},
		},
	}
	list, err := Validate([]v1alpha1.Schedule{ok})
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Name != "Daily restart" || list[0].Cron != "0 4 * * *" ||
		list[0].Tasks[0].Payload != "say Restart in 5 minutes" ||
		list[0].Tasks[1].Payload != "" {
		t.Errorf("normalized = %+v", list[0])
	}
	bad := map[string]func(s *v1alpha1.Schedule){
		"invalid cron":      func(s *v1alpha1.Schedule) { s.Cron = "every day" },
		"time zone in cron": func(s *v1alpha1.Schedule) { s.Cron = "CRON_TZ=UTC 0 4 * * *" },
		"@every":            func(s *v1alpha1.Schedule) { s.Cron = "@every 5m" },
		"empty name":        func(s *v1alpha1.Schedule) { s.Name = " " },
		"no tasks":          func(s *v1alpha1.Schedule) { s.Tasks = nil },
		"empty command":     func(s *v1alpha1.Schedule) { s.Tasks = []v1alpha1.ScheduleTask{{Action: "command"}} },
		"unknown action":    func(s *v1alpha1.Schedule) { s.Tasks = []v1alpha1.ScheduleTask{{Action: "reinstall"}} },
		"delay out of range": func(s *v1alpha1.Schedule) {
			s.Tasks = []v1alpha1.ScheduleTask{{Action: "stop", DelaySeconds: 901}}
		},
		"cron and event": func(s *v1alpha1.Schedule) { s.Event = EventStarted },
		"unknown event":  func(s *v1alpha1.Schedule) { s.Cron, s.Event = "", "crashed" },
		"power action at event": func(s *v1alpha1.Schedule) {
			s.Cron, s.Event, s.Tasks = "", EventStopping, []v1alpha1.ScheduleTask{{Action: "stop"}}
		},
		"stop tasks too long": func(s *v1alpha1.Schedule) {
			s.Cron, s.Event = "", EventStopping
			s.Tasks = []v1alpha1.ScheduleTask{
				{Action: "command", Payload: "say bye", DelaySeconds: 200},
				{Action: "command", Payload: "save-all", DelaySeconds: 200},
			}
		},
	}
	for name, change := range bad {
		s := ok
		s.Tasks = append([]v1alpha1.ScheduleTask(nil), ok.Tasks...)
		change(&s)
		if _, err := Validate([]v1alpha1.Schedule{s}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	backup := ok
	backup.Tasks = []v1alpha1.ScheduleTask{{Action: "backup", Payload: "nightly"}}
	if list, err := Validate([]v1alpha1.Schedule{backup}); err != nil || list[0].Tasks[0].Payload != "nightly" {
		t.Errorf("backup task: %v %+v", err, list)
	}
	dup := ok
	dup.Name = "daily RESTART"
	if _, err := Validate([]v1alpha1.Schedule{ok, dup}); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("duplicate names: %v", err)
	}
}

func TestRunnerRun(t *testing.T) {
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "srv", Namespace: tenancy.Namespace("alice")},
		Spec:       v1alpha1.GameServerSpec{State: v1alpha1.PowerRunning},
		Status:     v1alpha1.GameServerStatus{Phase: v1alpha1.PhaseOffline},
	}
	c := testutil.Builder(t).WithStatusSubresource(&v1alpha1.GameServer{}).WithObjects(gs).Build()
	log := testutil.Logger()
	triggered := 0
	r := &Runner{
		Client:   c,
		Reader:   c,
		Ops:      &serverctl.Ops{Client: c, Hub: console.NewHub(nil, log), Namespace: testutil.Namespace},
		Hub:      console.NewHub(nil, log),
		Trigger:  func(string, string) { triggered++ },
		Location: time.UTC,
		Log:      log,
	}
	ref := types.NamespacedName{Namespace: gs.Namespace, Name: gs.Name}
	result := func() string {
		cur := &v1alpha1.GameServer{}
		_ = c.Get(t.Context(), client.ObjectKeyFromObject(gs), cur)
		for _, s := range cur.Status.Schedules {
			return s.LastResult
		}
		return ""
	}

	r.run(
		t.Context(),
		ref,
		v1alpha1.Schedule{Name: "nightly", OnlyWhenOnline: true, Tasks: []v1alpha1.ScheduleTask{{Action: "stop"}}},
	)
	if got := result(); got != "skipped: server offline" {
		t.Errorf("offline server with onlyWhenOnline: %q", got)
	}
	r.run(t.Context(), ref, v1alpha1.Schedule{Name: "nightly", Tasks: []v1alpha1.ScheduleTask{{Action: "stop"}}})
	cur := &v1alpha1.GameServer{}
	_ = c.Get(t.Context(), client.ObjectKeyFromObject(gs), cur)
	if cur.Spec.State != v1alpha1.PowerStopped || triggered != 1 || result() != "ok" || len(cur.Status.Schedules) != 1 {
		t.Errorf(
			"power task: state=%s triggered=%d result=%q statuses=%d", cur.Spec.State, triggered, result(),
			len(cur.Status.Schedules),
		)
	}
	r.run(
		t.Context(),
		ref,
		v1alpha1.Schedule{Name: "nightly", Tasks: []v1alpha1.ScheduleTask{{Action: "command", Payload: "say hi"}}},
	)
	if got := result(); !strings.HasPrefix(got, "failed: task 1 (command): server is not running") {
		t.Errorf("command on an offline server: %q", got)
	}
}

func TestValidateEventSchedules(t *testing.T) {
	list, err := Validate([]v1alpha1.Schedule{
		{
			Name:           "Welcome",
			Event:          EventStarted,
			Enabled:        true,
			OnlyWhenOnline: true,
			Tasks:          []v1alpha1.ScheduleTask{{Action: "command", Payload: "say online"}},
		},
		{Name: "Goodbye", Event: EventStopping, Enabled: true, Tasks: []v1alpha1.ScheduleTask{
			{
				Action:  "command",
				Payload: "say Stopping in 30 seconds",
			},
			{Action: "command", Payload: "save-all", DelaySeconds: 30},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if list[0].OnlyWhenOnline || list[0].Cron != "" {
		t.Errorf("event schedules have no cron and ignore onlyWhenOnline: %+v", list[0])
	}
}

// The stop waits while the "stopping" tasks run: Stopping is true right after StartStopping.
func TestStoppingTasks(t *testing.T) {
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "srv", Namespace: tenancy.Namespace("alice")},
		Spec: v1alpha1.GameServerSpec{Schedules: []v1alpha1.Schedule{
			{
				Name:    "Goodbye",
				Event:   EventStopping,
				Enabled: true,
				Tasks:   []v1alpha1.ScheduleTask{{Action: "command", Payload: "say bye", DelaySeconds: 1}},
			},
			{
				Name:  "Disabled",
				Event: EventStopping,
				Tasks: []v1alpha1.ScheduleTask{{Action: "command", Payload: "say never", DelaySeconds: 60}},
			},
		}},
	}
	c := testutil.Builder(t).WithStatusSubresource(&v1alpha1.GameServer{}).WithObjects(gs).Build()
	log := testutil.Logger()
	r := &Runner{
		Client:   c,
		Reader:   c,
		Ops:      &serverctl.Ops{Client: c, Hub: console.NewHub(nil, log)},
		Hub:      console.NewHub(nil, log),
		Trigger:  func(string, string) {},
		Location: time.UTC,
		Log:      log,
	}

	wait := r.StartStopping(gs)
	if wait < 26*time.Second || wait > 60*time.Second {
		t.Errorf("wait = %s, want the delay plus margins of the enabled schedule only", wait)
	}
	if !r.Stopping(gs) {
		t.Fatal("Stopping must be true right after StartStopping")
	}
	deadline := time.Now().Add(10 * time.Second)
	for r.Stopping(gs) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if r.Stopping(gs) {
		t.Error("the tasks did not finish")
	}
	empty := &v1alpha1.GameServer{ObjectMeta: gs.ObjectMeta}
	if r.StartStopping(empty) != 0 {
		t.Error("a server without stop tasks must not wait")
	}
}
