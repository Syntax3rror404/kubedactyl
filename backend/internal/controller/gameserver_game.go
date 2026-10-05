package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"app/api/v1alpha1"
	"app/internal/console"
	"app/internal/files"
	"app/internal/gameserver"
)

// crashWindow: a second crash within this time does not trigger another restart.
const crashWindow = 60 * time.Second

// reconcileGame drives the game pod towards the desired power state: it follows a pod that
// runs, removes one whose exit was handled, and otherwise reports offline or starts the server.
func (r *Reconciler) reconcileGame(
	ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg,
) (reconcile.Result, error) {
	// A suspended server is stopped and stays stopped.
	wantRunning := gs.Spec.State == v1alpha1.PowerRunning && !gs.Spec.Suspended
	restart := gs.Annotations[gameserver.AnnotationRestart] != "" && !gs.Spec.Suspended

	pod := &corev1.Pod{}
	err := r.Reader.Get(ctx, types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.GamePodName(gs.Name)}, pod)
	switch {
	case err != nil && !apierrors.IsNotFound(err):
		return reconcile.Result{}, err
	case err == nil && pod.Annotations[gameserver.AnnotationExitHandled] == "":
		return r.reconcileGamePod(ctx, gs, egg, pod, wantRunning, restart)
	case err == nil:
		if res, done, err := r.removeExitedPod(ctx, pod, wantRunning || restart); done {
			return res, err
		}
	}

	// No game pod.
	if err := r.clearPodState(ctx, gs); err != nil {
		return reconcile.Result{}, err
	}
	if restart {
		if err := r.clearRestart(ctx, gs); err != nil {
			return reconcile.Result{}, err
		}
		wantRunning = true
	}
	if !wantRunning {
		r.setPhase(gs, v1alpha1.PhaseOffline, offlineMessage(gs))
		return reconcile.Result{}, nil
	}
	return r.startGame(ctx, gs, egg)
}

// removeExitedPod deletes a game pod whose exit was already handled: right away when the server
// should start again, otherwise after the grace period that keeps its output readable.
// done is true while the caller has to wait for the pod to go.
func (r *Reconciler) removeExitedPod(
	ctx context.Context, pod *corev1.Pod, start bool,
) (res reconcile.Result, done bool, err error) {
	if start && pod.DeletionTimestamp == nil {
		if err := r.Delete(ctx, pod, client.GracePeriodSeconds(0)); client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, true, err
		}
	} else if wait, err := r.removeTerminated(ctx, pod); err != nil {
		return reconcile.Result{}, true, err
	} else if wait > 0 {
		return reconcile.Result{RequeueAfter: wait}, true, nil
	}
	if start || pod.DeletionTimestamp != nil {
		return reconcile.Result{RequeueAfter: time.Second}, true, nil
	}
	return reconcile.Result{}, false, nil
}

// clearPodState resets the status fields of the previous game pod.
func (r *Reconciler) clearPodState(ctx context.Context, gs *v1alpha1.GameServer) error {
	gs.Status.RestartRequired = false
	if gs.Status.PodUID != "" {
		// The pod vanished without passing handleExit, e.g. after a kill.
		r.Hub.Daemon(gs.Name, "Server marked as offline...")
	}
	gs.Status.PodUID = ""
	gs.Status.StopRequestedAt = nil
	gs.Status.StopTasksStartedAt = nil
	if _, ok := gs.Annotations[gameserver.AnnotationStopSent]; !ok {
		return nil
	}
	patch := client.MergeFrom(gs.DeepCopy())
	delete(gs.Annotations, gameserver.AnnotationStopSent)
	status := gs.Status // Patch overwrites the status with the stored one
	err := r.Patch(ctx, gs, patch)
	gs.Status = status
	return err
}

// offlineMessage keeps a crash message visible until the server is started again.
func offlineMessage(gs *v1alpha1.GameServer) string {
	switch {
	case gs.Spec.Suspended:
		return "suspended by an administrator"
	case strings.HasPrefix(gs.Status.Message, "crashed"):
		return gs.Status.Message
	}
	return ""
}

// startGame runs the pre-start steps in the files pod (started for them if needed) and creates
// the game pod.
func (r *Reconciler) startGame(
	ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg,
) (reconcile.Result, error) {
	ref := files.RefOf(gs)
	r.Files.Touch(ref)
	filesReady, err := r.ensureFilesPod(ctx, gs)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("files pod: %w", err)
	}
	if !filesReady {
		r.setPhase(gs, v1alpha1.PhaseStarting, "starting the file container")
		return reconcile.Result{RequeueAfter: 2 * time.Second}, nil
	}

	r.Hub.Daemon(gs.Name, "Server marked as starting...")
	r.Hub.Daemon(gs.Name, "Updating process configuration files...")
	preCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	warnings, err := gameserver.PreStart(preCtx, r.Kube, gs, egg, r.Opts)
	cancel()
	r.Files.Touch(ref)
	for _, w := range warnings {
		r.Hub.Daemon(gs.Name, w)
	}
	if err != nil {
		r.Hub.Daemon(gs.Name, "Pre-start failed: "+err.Error())
		r.setPhase(gs, v1alpha1.PhaseStarting, "pre-start failed: "+err.Error())
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	want := gameserver.GamePod(gs, egg, r.Opts)
	if err := controllerutil.SetControllerReference(gs, want, r.Scheme()); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.Create(ctx, want); client.IgnoreAlreadyExists(err) != nil {
		return reconcile.Result{}, err
	}
	now := metav1.Now()
	gs.Status.StartedAt = &now
	gs.Status.LastExitCode = nil
	r.setPhase(gs, v1alpha1.PhaseStarting, "")
	return reconcile.Result{}, nil
}

// reconcileGamePod follows a game pod that has not exited (or whose exit is not handled yet):
// exit handling, stopping, start problems and the running detection.
func (r *Reconciler) reconcileGamePod(
	ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg, pod *corev1.Pod, wantRunning, restart bool,
) (reconcile.Result, error) {
	r.trackPod(gs, egg, pod)
	if done, code, reason := terminated(pod); done {
		return r.handleExit(ctx, gs, pod, code, reason, wantRunning, restart)
	}
	switch {
	case pod.DeletionTimestamp != nil:
		r.setPhase(gs, v1alpha1.PhaseStopping, "")
		return reconcile.Result{RequeueAfter: 2 * time.Second}, nil
	case !wantRunning || restart:
		return r.stopGamePod(ctx, gs, egg, pod)
	}
	if problem := waitingProblem(pod); problem != "" {
		r.setPhase(gs, v1alpha1.PhaseStarting, problem)
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}
	// Running once the egg's "done" output appeared (or right away for eggs without one).
	if gs.Status.Phase == v1alpha1.PhaseRunning || r.Hub.IsDone(pod.UID) ||
		(len(egg.Spec.StartupDone) == 0 && pod.Status.Phase == corev1.PodRunning) {
		if gs.Status.Phase != v1alpha1.PhaseRunning {
			r.Hub.Daemon(gs.Name, "Server marked as running...")
			if r.Events != nil {
				r.Events.Started(gs)
			}
		}
		r.setPhase(gs, v1alpha1.PhaseRunning, "")
		return reconcile.Result{}, nil
	}
	r.setPhase(gs, v1alpha1.PhaseStarting, "")
	return reconcile.Result{}, nil
}

// trackPod records a new pod in the status, follows its output and compares its runtime hash.
func (r *Reconciler) trackPod(gs *v1alpha1.GameServer, egg *v1alpha1.Egg, pod *corev1.Pod) {
	if gs.Status.PodUID != string(pod.UID) {
		gs.Status.PodUID = string(pod.UID)
		gs.Status.StartedAt = &pod.CreationTimestamp
		gs.Status.StopRequestedAt = nil
		gs.Status.StopTasksStartedAt = nil
		if gs.Status.Phase == v1alpha1.PhaseRunning {
			gs.Status.Phase = v1alpha1.PhaseStarting
		}
	}
	r.Hub.Follow(gs.Name, pod, console.KindGame, console.NewMatcher(egg.Spec.StartupDone, egg.Spec.StripAnsi))
	// Pods created before the hash existed have none and never ask for a restart.
	hash, ok := pod.Annotations[gameserver.AnnotationRuntimeHash]
	gs.Status.RestartRequired = ok && hash != gameserver.RuntimeHash(gs, egg, r.Opts)
}

// handleExit processes a game pod whose process exited: normal stop, restart or crash.
func (r *Reconciler) handleExit(
	ctx context.Context, gs *v1alpha1.GameServer, pod *corev1.Pod, code int32, reason string, wantRunning, restart bool,
) (reconcile.Result, error) {
	// A pod that is being deleted was stopped on purpose (kill, reinstall), not crashed.
	stopRequested := gs.Status.StopRequestedAt != nil || pod.DeletionTimestamp != nil
	gs.Status.LastExitCode = ptr.To(code)
	if pod.DeletionTimestamp == nil {
		patch := client.MergeFrom(pod.DeepCopy())
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		pod.Annotations[gameserver.AnnotationExitHandled] = "true"
		if err := r.Patch(ctx, pod, patch); client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}
	}
	r.Hub.ForgetPod(pod.UID)
	gs.Status.PodUID = ""
	gs.Status.StopRequestedAt = nil
	gs.Status.StopTasksStartedAt = nil

	if restart {
		r.Hub.Daemon(gs.Name, "Server marked as offline, restarting...")
		r.setPhase(gs, v1alpha1.PhaseOffline, "")
		return reconcile.Result{RequeueAfter: time.Second}, nil
	}
	if stopRequested || !wantRunning {
		r.Hub.Daemon(gs.Name, "Server marked as offline...")
		r.setPhase(gs, v1alpha1.PhaseOffline, "")
		return reconcile.Result{}, nil
	}
	return r.handleCrash(ctx, gs, code, reason)
}

// handleCrash processes a process that exited on its own (a clean exit without a stop request
// counts, too): restart once, stop the server when it crashes again within crashWindow.
func (r *Reconciler) handleCrash(
	ctx context.Context, gs *v1alpha1.GameServer, code int32, reason string,
) (reconcile.Result, error) {
	detail := fmt.Sprintf("exit code %d", code)
	if reason == "OOMKilled" {
		detail += ", out of memory"
	}
	r.Hub.Daemon(gs.Name, "---------- Detected server process in a crashed state! ----------")
	r.Hub.Daemon(gs.Name, "Exit code: "+detail)
	lastCrash := gs.Status.LastCrashAt
	now := metav1.Now()
	gs.Status.LastCrashAt = &now
	crashRestart := gs.Spec.CrashRestart == nil || *gs.Spec.CrashRestart
	if crashRestart && (lastCrash == nil || time.Since(lastCrash.Time) > crashWindow) {
		r.Hub.Daemon(gs.Name, "Aborting automatic restart in 60 seconds if the server crashes again...")
		r.setPhase(gs, v1alpha1.PhaseOffline, "crashed ("+detail+"), restarting")
		return reconcile.Result{RequeueAfter: time.Second}, nil
	}
	if crashRestart {
		r.Hub.Daemon(gs.Name, "Aborting automatic restart, last crash occurred less than 60 seconds ago.")
	}
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.State = v1alpha1.PowerStopped
	status := gs.Status
	if err := r.Patch(ctx, gs, patch); err != nil {
		return reconcile.Result{}, err
	}
	gs.Status = status
	r.setPhase(gs, v1alpha1.PhaseOffline, "crashed ("+detail+")")
	return reconcile.Result{}, nil
}

// clearRestart removes the restart request and keeps the server wanted running (the next
// reconcile creates the new pod).
func (r *Reconciler) clearRestart(ctx context.Context, gs *v1alpha1.GameServer) error {
	patch := client.MergeFrom(gs.DeepCopy())
	delete(gs.Annotations, gameserver.AnnotationRestart)
	gs.Spec.State = v1alpha1.PowerRunning
	status := gs.Status
	err := r.Patch(ctx, gs, patch)
	gs.Status = status
	return err
}
