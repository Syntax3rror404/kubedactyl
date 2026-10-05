package selfupgrade

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/internal/testutil"
)

func TestNewerVersions(t *testing.T) {
	tags := []string{"0.2.1", "0.2.3", "0.3.0-rc.1", "0.2.10", "latest", "0.2.2"}
	latest, newer := newerVersions(tags, "0.2.2")
	if latest != "0.2.10" {
		t.Errorf("latest = %q, want 0.2.10 (numeric order, no pre-release)", latest)
	}
	if !slices.Equal(newer, []string{"0.2.10", "0.2.3"}) {
		t.Errorf("newer = %v", newer)
	}
	if _, newer := newerVersions(tags, "dev"); len(newer) != 0 {
		t.Errorf("a dev build must not offer upgrades: %v", newer)
	}
}

func TestCheckerKeepsErrors(t *testing.T) {
	c := &Checker{
		Config: Config{Chart: "oci://example.com/charts/kubedactyl", Current: "0.2.2"},
		Log:    testutil.Logger(),
	}
	c.tags = func(ref string) ([]string, error) {
		if ref != "example.com/charts/kubedactyl" {
			t.Errorf("ref = %q", ref)
		}
		return nil, errors.New("offline")
	}
	if got := c.Refresh(); got.Error != "offline" || got.Latest != "" || c.Last().CheckedAt.IsZero() {
		t.Errorf("check = %+v", got)
	}
}

func TestJobFor(t *testing.T) {
	cfg := Config{Chart: "oci://ghcr.io/x/charts/kubedactyl", Release: "kubedactyl", Namespace: "kubedactyl",
		ServiceAccount: "kubedactyl-upgrader", Image: "ghcr.io/x/kubedactyl:0.2.2", Current: "0.2.2"}
	job := JobFor(cfg, "0.2.3")
	pod := job.Spec.Template.Spec
	if pod.ServiceAccountName != "kubedactyl-upgrader" || pod.Containers[0].Image != cfg.Image {
		t.Errorf("service account %q, image %q", pod.ServiceAccountName, pod.Containers[0].Image)
	}
	want := []string{
		"upgrade",
		"--release",
		"kubedactyl",
		"--release-namespace",
		"kubedactyl",
		"--chart",
		cfg.Chart,
		"--version",
		"0.2.3",
	}
	if !slices.Equal(pod.Containers[0].Args, want) {
		t.Errorf("args = %v", pod.Containers[0].Args)
	}
	if *job.Spec.BackoffLimit != 0 || job.Labels[labelVersion] != "0.2.3" ||
		job.Annotations[annotationFrom] != "0.2.2" {
		t.Errorf("job metadata/spec: %+v", job.ObjectMeta)
	}
	if j := jobOf(job); j.State != "running" || j.Version != "0.2.3" || j.From != "0.2.2" {
		t.Errorf("job view = %+v", j)
	}
}

func TestParseArgs(t *testing.T) {
	if _, err := ParseArgs(
		[]string{
			"--release",
			"k",
			"--release-namespace",
			"k",
			"--chart",
			"oci://x/y",
			"--chart-file",
			"c.tgz",
			"--version",
			"1.0.0",
		},
	); err == nil {
		t.Error("--chart and --chart-file are exclusive")
	}
	if _, err := ParseArgs(
		[]string{"--release", "k", "--release-namespace", "k", "--chart", "oci://x/y", "--version", "latest"},
	); err == nil {
		t.Error("a version must be a semantic version")
	}
	o, err := ParseArgs(
		[]string{"--release", "k", "--release-namespace", "ns", "--chart", "oci://x/y", "--version", "1.2.3"},
	)
	if err != nil || o.Namespace != "ns" || o.Version != "1.2.3" {
		t.Errorf("options %+v, err %v", o, err)
	}
}

// SELFUPGRADE_IT_CHART=oci://ghcr.io/syntax3rror404/charts/kubedactyl go test ./internal/selfupgrade/ -run Registry -v
func TestRegistryTags(t *testing.T) {
	chart := os.Getenv("SELFUPGRADE_IT_CHART")
	if chart == "" {
		t.Skip("set SELFUPGRADE_IT_CHART")
	}
	c := &Checker{Config: Config{Chart: chart, Current: "0.2.1"}, Log: testutil.Logger()}
	res := c.Refresh()
	if res.Error != "" || res.Latest == "" {
		t.Fatalf("check = %+v", res)
	}
	t.Logf("latest %s, newer than 0.2.1: %v", res.Latest, res.Newer)
	rc, err := newRegistryClient()
	if err != nil {
		t.Fatal(err)
	}
	ch, src, err := loadChart(rc, RunOptions{Chart: chart, Version: res.Latest})
	if err != nil || ch.Metadata.Version != res.Latest || ch.Metadata.Name != "kubedactyl" {
		t.Fatalf("pull %s: %v %+v", src, err, ch)
	}
}

func TestPruneKeepsTheNewestJobs(t *testing.T) {
	now := time.Now()
	job := func(name string, age time.Duration, done bool) *batchv1.Job {
		j := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "kubedactyl", CreationTimestamp: metav1.NewTime(now.Add(-age)),
			Labels: map[string]string{labelComponent: componentValue},
		}}
		if done {
			j.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}
		}
		return j
	}
	c := testutil.Builder(t).WithObjects(
		job("newest", time.Minute, true), job("second", time.Hour, true), job("third", 2*time.Hour, false),
		job("fourth", 3*time.Hour, true), job("fifth", 4*time.Hour, true), job("old-running", 5*time.Hour, false),
	).Build()
	u := &Upgrader{Config: Config{Namespace: "kubedactyl"}, Client: c, Reader: c}
	if err := u.Prune(context.Background()); err != nil {
		t.Fatal(err)
	}
	var list batchv1.JobList
	_ = c.List(context.Background(), &list)
	var names []string
	for _, j := range list.Items {
		names = append(names, j.Name)
	}
	slices.Sort(names)
	if want := []string{"newest", "old-running", "second", "third"}; !slices.Equal(names, want) {
		t.Errorf("kept %v, want %v", names, want)
	}
}
