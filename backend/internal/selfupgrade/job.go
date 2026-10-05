package selfupgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"app/internal/kube"
)

const (
	labelComponent = "app.kubernetes.io/component"
	componentValue = "self-upgrade"
	labelVersion   = "kubedactyl.io/upgrade-version"
	annotationFrom = "kubedactyl.io/upgrade-from"
)

// ErrRunning is returned while another upgrade job is active.
var ErrRunning = errors.New("an upgrade is already running")

// KeepJobs is how many upgrade jobs (with their pods) stay as history; older finished ones
// are deleted after every upgrade and once an hour.
const KeepJobs = 3

// Job states.
const (
	JobRunning   = "running"
	JobSucceeded = "succeeded"
	JobFailed    = "failed"
)

// Job is an upgrade job as shown in the web interface.
type Job struct {
	Name    string `json:"name"`
	From    string `json:"from"`
	Version string `json:"version"`
	// State is running, succeeded or failed.
	State      string     `json:"state"                enums:"running,succeeded,failed" example:"running"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	// Log holds the last lines of the job output (for failed jobs).
	Log string `json:"log,omitempty"`
}

// Upgrader starts upgrade jobs and reports their state.
type Upgrader struct {
	Config Config
	Client client.Client
	// Reader reads the jobs uncached (their state changes while the page polls).
	Reader client.Reader
	Kube   *kube.Client
}

// Start creates the job that upgrades the release to version.
func (u *Upgrader) Start(ctx context.Context, version string) (*Job, error) {
	jobs, err := u.Jobs(ctx, false)
	if err != nil {
		return nil, err
	}
	if slices.ContainsFunc(jobs, func(j Job) bool { return j.State == JobRunning }) {
		return nil, ErrRunning
	}
	job := JobFor(u.Config, version)
	if err := u.Client.Create(ctx, job); err != nil {
		return nil, err
	}
	_ = u.Prune(ctx) // best effort: the hourly run tries again
	j := jobOf(job)
	return &j, nil
}

// Prune deletes finished upgrade jobs beyond the newest KeepJobs, together with their pods.
// Running jobs are never deleted.
func (u *Upgrader) Prune(ctx context.Context) error {
	var list batchv1.JobList
	if err := u.Reader.List(
		ctx, &list, client.InNamespace(u.Config.Namespace), client.MatchingLabels{labelComponent: componentValue},
	); err != nil {
		return err
	}
	jobs := list.Items
	sort.Slice(jobs, func(a, b int) bool { return jobs[a].CreationTimestamp.After(jobs[b].CreationTimestamp.Time) })
	for i := KeepJobs; i < len(jobs); i++ {
		if jobOf(&jobs[i]).State == JobRunning {
			continue
		}
		err := u.Client.Delete(ctx, &jobs[i], client.PropagationPolicy(metav1.DeletePropagationBackground))
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

// PruneEvery runs Prune at the start and then every interval (a manager runnable).
func (u *Upgrader) PruneEvery(interval time.Duration, log *slog.Logger) manager.RunnableFunc {
	return func(ctx context.Context) error {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			if err := u.Prune(ctx); err != nil {
				log.Warn("removing old upgrade jobs", "err", err)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
			}
		}
	}
}

// Jobs lists the upgrade jobs, newest first; withLog adds the output of failed jobs.
func (u *Upgrader) Jobs(ctx context.Context, withLog bool) ([]Job, error) {
	var list batchv1.JobList
	if err := u.Reader.List(
		ctx, &list, client.InNamespace(u.Config.Namespace), client.MatchingLabels{labelComponent: componentValue},
	); err != nil {
		return nil, err
	}
	out := make([]Job, 0, len(list.Items))
	for i := range list.Items {
		j := jobOf(&list.Items[i])
		if withLog && j.State == JobFailed {
			j.Log = u.jobLog(ctx, list.Items[i].Name)
		}
		out = append(out, j)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].StartedAt.After(out[b].StartedAt) })
	return out, nil
}

// jobLog returns the last lines of the job's pod output.
func (u *Upgrader) jobLog(ctx context.Context, job string) string {
	var pods corev1.PodList
	if err := u.Reader.List(
		ctx, &pods, client.InNamespace(u.Config.Namespace), client.MatchingLabels{"job-name": job},
	); err != nil ||
		len(pods.Items) == 0 {
		return ""
	}
	pod := &pods.Items[0]
	req := u.Kube.Clientset.CoreV1().
		Pods(u.Config.Namespace).
		GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: ptr.To[int64](20)})
	if data, err := req.DoRaw(ctx); err == nil && len(bytes.TrimSpace(data)) > 0 {
		return strings.TrimSpace(string(bytes.ToValidUTF8(data, nil)))
	}
	// No output: the container did not start (image, command).
	for _, cs := range pod.Status.ContainerStatuses {
		if t := cs.State.Terminated; t != nil {
			return strings.TrimSpace(t.Reason + ": " + t.Message)
		}
		if w := cs.State.Waiting; w != nil {
			return strings.TrimSpace(w.Reason + ": " + w.Message)
		}
	}
	return ""
}

func jobOf(job *batchv1.Job) Job {
	j := Job{
		Name: job.Name, From: job.Annotations[annotationFrom], Version: job.Labels[labelVersion],
		State: JobRunning, StartedAt: job.CreationTimestamp.Time,
	}
	if j.StartedAt.IsZero() {
		j.StartedAt = time.Now()
	}
	for _, c := range job.Status.Conditions {
		if c.Status != corev1.ConditionTrue {
			continue
		}
		switch c.Type {
		case batchv1.JobComplete:
			j.State = JobSucceeded
		case batchv1.JobFailed:
			j.State = JobFailed
		default:
			continue
		}
		t := c.LastTransitionTime.Time
		j.FinishedAt = &t
	}
	return j
}

// JobFor builds the upgrade job. It runs this panel's image with the upgrade service account;
// the job is not part of the release, so the upgrade does not remove it.
func JobFor(c Config, version string) *batchv1.Job {
	labels := map[string]string{
		"app.kubernetes.io/name": "kubedactyl", "app.kubernetes.io/instance": c.Release,
		labelComponent: componentValue, labelVersion: version,
	}
	tmp := corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: ptr.To(resource.MustParse("64Mi"))},
		},
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-upgrade-%s-", c.Release, strings.ReplaceAll(version, ".", "-")),
			Namespace:    c.Namespace, Labels: labels,
			Annotations: map[string]string{annotationFrom: c.Current},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            ptr.To[int32](0),
			ActiveDeadlineSeconds:   ptr.To[int64](600),
			TTLSecondsAfterFinished: ptr.To[int32](24 * 3600),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: c.ServiceAccount,
					RestartPolicy:      corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true), RunAsUser: ptr.To[int64](65532), RunAsGroup: ptr.To[int64](65532),
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Volumes: []corev1.Volume{tmp},
					Containers: []corev1.Container{{
						Name:  "upgrade",
						Image: c.Image,
						Args: []string{"upgrade", "--release", c.Release, "--release-namespace", c.Namespace,
							"--chart", c.Chart, "--version", version},
						// Helm keeps its caches under HOME.
						Env:          []corev1.EnvVar{{Name: "HOME", Value: "/tmp"}},
						VolumeMounts: []corev1.VolumeMount{{Name: "tmp", MountPath: "/tmp"}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false), ReadOnlyRootFilesystem: ptr.To(true),
							Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
					}},
				},
			},
		},
	}
}
