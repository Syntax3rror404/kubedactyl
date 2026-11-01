package controller

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
)

var running = corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}

// A reinstall of a running server stops it through the normal stop path first; the install starts
// once no game pod is tracked any more.
func TestReconcileReinstall(t *testing.T) {
	for _, tc := range []struct {
		name      string
		pod       bool
		wantPhase v1alpha1.Phase
		wantMsg   string
	}{
		{"running server is stopped first", true, v1alpha1.PhaseStopping, ""},
		{"stopped server installs", false, v1alpha1.PhaseInstalling, "waiting for the volume to be provisioned"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := testServer()
			gs.Spec.InstallRevision = 2
			gs.Status = v1alpha1.GameServerStatus{InstalledRevision: 1, Phase: v1alpha1.PhaseRunning}
			objects := []client.Object{gs, testEgg()}
			if tc.pod {
				pod := gamePod("pod-1", running)
				gs.Status.PodUID = string(pod.UID)
				objects = append(objects, pod)
			}
			r, _ := testReconciler(t, nil, objects...)
			if _, err := r.reconcile(context.Background(), gs); err != nil {
				t.Fatal(err)
			}
			if gs.Status.Phase != tc.wantPhase || gs.Status.Message != tc.wantMsg {
				t.Errorf("phase %q (%q), want %q (%q)", gs.Status.Phase, gs.Status.Message, tc.wantPhase, tc.wantMsg)
			}
			if tc.pod && gs.Status.StopRequestedAt == nil {
				t.Error("the stop was not sent")
			}
		})
	}
}

// Without a game pod the install clears what the status says about the old pod.
func TestStopGameForInstallClearsPodState(t *testing.T) {
	gs := testServer()
	gs.Status.RestartRequired, gs.Status.StopRequestedAt = true, &metav1.Time{Time: time.Now()}
	r, _ := testReconciler(t, nil, gs)
	_, done, err := r.stopGameForInstall(context.Background(), gs)
	if err != nil || done {
		t.Fatalf("done = %v, %v", done, err)
	}
	if gs.Status.RestartRequired || gs.Status.StopRequestedAt != nil {
		t.Errorf("pod state kept: %+v", gs.Status)
	}
}

// A pod whose container never ran is deleted at once on a stop; a running one gets its stop.
func TestStopGamePod(t *testing.T) {
	pulling := corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}
	for _, tc := range []struct {
		name          string
		state         corev1.ContainerState
		wantRequested bool
	}{
		{"image cannot be pulled", pulling, false},
		{"not scheduled yet", corev1.ContainerState{}, false},
		{"running", running, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := testServer()
			pod := gamePod("pod-1", tc.state)
			if tc.state == (corev1.ContainerState{}) {
				pod.Status.ContainerStatuses = nil
			}
			r, c := testReconciler(t, nil, gs, pod)
			if _, err := r.stopGamePod(context.Background(), gs, testEgg(), pod); err != nil {
				t.Fatal(err)
			}
			if (gs.Status.StopRequestedAt != nil) != tc.wantRequested || gs.Status.Phase != v1alpha1.PhaseStopping {
				t.Errorf("stop requested %v, phase %q", gs.Status.StopRequestedAt != nil, gs.Status.Phase)
			}
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(pod), pod); !apierrors.IsNotFound(err) {
				t.Errorf("pod still there: %v", err)
			}
		})
	}
}

// After preStartTries failures in a row the server is stopped; a stop in between starts the count over.
func TestPreStartFailures(t *testing.T) {
	ctx := context.Background()
	gs := testServer()
	r, c := testReconciler(t, nil, gs)
	fail := errors.New("config file broken")
	for i, stop := range []bool{false, false, true, false, false} {
		if stop {
			gs.Spec.State = v1alpha1.PowerStopped
			if _, err := r.reconcileGame(ctx, gs, testEgg()); err != nil {
				t.Fatal(err)
			}
			gs.Spec.State = v1alpha1.PowerRunning
			continue
		}
		if _, err := r.handlePreStartFailure(ctx, gs, fail); err != nil {
			t.Fatal(err)
		}
		if gs.Status.Phase != v1alpha1.PhaseStarting || !strings.HasPrefix(gs.Status.Message, preStartFailed) {
			t.Fatalf("failure %d: phase %q (%q), want a retry", i+1, gs.Status.Phase, gs.Status.Message)
		}
	}
	if _, err := r.handlePreStartFailure(ctx, gs, fail); err != nil {
		t.Fatal(err)
	}
	stored := &v1alpha1.GameServer{}
	_ = c.Get(ctx, client.ObjectKeyFromObject(gs), stored)
	if gs.Status.Phase != v1alpha1.PhaseOffline || stored.Spec.State != v1alpha1.PowerStopped {
		t.Errorf("third failure in a row: phase %q, state %q; want offline and stopped", gs.Status.Phase,
			stored.Spec.State)
	}
	if offlineMessage(gs) != gs.Status.Message {
		t.Errorf("the reason must stay visible: %q", offlineMessage(gs))
	}
}

// fakeEvents reports whether the "stopping" tasks still run and how long the stop may wait.
type fakeEvents struct {
	running bool
	limit   time.Duration
}

func (e fakeEvents) Started(*v1alpha1.GameServer)                     {}
func (e fakeEvents) StartStopping(*v1alpha1.GameServer) time.Duration { return e.limit }
func (e fakeEvents) Stopping(*v1alpha1.GameServer) bool               { return e.running }
func (e fakeEvents) StopLimit(*v1alpha1.GameServer) time.Duration     { return e.limit }

// The stop waits for the "stopping" tasks while they run, at most StopLimit.
func TestRunStopTasks(t *testing.T) {
	started := func(ago time.Duration) *metav1.Time { return &metav1.Time{Time: time.Now().Add(-ago)} }
	for _, tc := range []struct {
		name     string
		events   fakeEvents
		started  *metav1.Time
		wantWait bool
	}{
		{"no tasks", fakeEvents{limit: 0}, nil, false},
		{"tasks start", fakeEvents{running: true, limit: time.Minute}, nil, true},
		{"tasks run", fakeEvents{running: true, limit: time.Minute}, started(10 * time.Second), true},
		{"tasks take too long", fakeEvents{running: true, limit: time.Minute}, started(2 * time.Minute), false},
		{"tasks done", fakeEvents{limit: time.Minute}, started(10 * time.Second), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := testServer()
			gs.Status.Phase, gs.Status.StopTasksStartedAt = v1alpha1.PhaseRunning, tc.started
			r, _ := testReconciler(t, nil, gs)
			r.Events = tc.events
			if wait := r.runStopTasks(gs, gamePod("pod-1", running)); (wait > 0) != tc.wantWait {
				t.Errorf("wait = %v, want waiting %v", wait, tc.wantWait)
			}
		})
	}
}

// The install pod goes once the log stream had time to deliver its last lines.
func TestFollowInstallDone(t *testing.T) {
	gs := testServer()
	gs.Spec.InstallRevision = 1
	pod := gamePod("install-1", corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}})
	pod.Name = gameserver.InstallPodName(gs.Name)
	r, _ := testReconciler(t, nil, gs, pod)
	res, err := r.followInstall(gs, pod)
	if err != nil || res.RequeueAfter != terminatedGrace || gs.Status.InstalledRevision != 1 {
		t.Errorf("result %+v, %v, installed revision %d", res, err, gs.Status.InstalledRevision)
	}
}
