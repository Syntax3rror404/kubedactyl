package files

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/testutil"
)

// TestForeignFilesPod: a pod with the files pod's name that the server does not control is never
// used for file operations.
func TestForeignFilesPod(t *testing.T) {
	ref := Ref{Namespace: "ns", Name: "srv"}
	ready := corev1.PodStatus{
		Phase:      corev1.PodRunning,
		Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
	}
	owner := metav1.OwnerReference{
		APIVersion: v1alpha1.GroupVersion.String(), Kind: "GameServer", Name: "srv", UID: "1", Controller: ptr.To(true),
	}
	for name, refs := range map[string][]metav1.OwnerReference{
		"no owner":       nil,
		"other server":   {func() metav1.OwnerReference { o := owner; o.Name = "other"; return o }()},
		"not controller": {func() metav1.OwnerReference { o := owner; o.Controller = nil; return o }()},
		"ours":           {owner},
	} {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ref.Namespace, Name: gameserver.FilesPodName(ref.Name), OwnerReferences: refs,
			},
			Status: ready,
		}
		s := &Service{
			Activity: NewActivity(), Reader: testutil.Builder(t).WithObjects(pod).Build(),
			Trigger: func(string, string) {},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := s.EnsurePod(ctx, ref)
		cancel()
		if want := name != "ours"; errors.Is(err, ErrForeignPod) != want || (!want && err != nil) {
			t.Errorf("%s: EnsurePod = %v, foreign expected %v", name, err, want)
		}
	}
}

// TestTrimJobs: only finished jobs are dropped, so a running backup still makes the server busy.
func TestTrimJobs(t *testing.T) {
	s := &Service{Activity: NewActivity()}
	ref := Ref{Namespace: "ns", Name: "srv"}
	if _, err := s.add(ref, JobBackup, "backup"); err != nil {
		t.Fatal(err)
	}
	for range keptJobs + 5 {
		j, err := s.add(ref, JobCompress, "archive")
		if err != nil {
			t.Fatal(err)
		}
		s.finish(j, nil)
	}
	if n := len(s.Jobs(ref)); n != keptJobs {
		t.Errorf("%d jobs kept, want %d", n, keptJobs)
	}
	if !s.Busy(ref, JobBackup) {
		t.Error("the running backup was dropped")
	}
	if _, err := s.add(ref, JobRestore, "restore"); !errors.Is(err, ErrBusy) {
		t.Errorf("restore during a backup: %v", err)
	}
}

// TestForget: a deleted server leaves no jobs and no file activity behind.
func TestForget(t *testing.T) {
	s := &Service{Activity: NewActivity()}
	ref := Ref{Namespace: "ns", Name: "srv"}
	if _, err := s.add(ref, JobBackup, "backup"); err != nil {
		t.Fatal(err)
	}
	s.Activity.Touch(ref)
	s.Forget(ref)
	if len(s.Jobs(ref)) != 0 || s.Busy(ref, JobBackup) || s.Activity.Active(ref) {
		t.Error("the server is still known")
	}
}
