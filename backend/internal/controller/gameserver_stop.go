package controller

import (
	"context"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"app/api/v1alpha1"
	"app/internal/gameserver"
)

// stopTaskLimits remembers per pod how long a stop may wait for its tasks (lost on a panel
// restart; then the stop is sent right away).
var stopTaskLimits sync.Map

// runStopTasks runs the "stopping" tasks of a running server before its stop is sent and
// returns how long to wait before looking again (0: send the stop now). A stop the user typed
// into the console is already on its way, so it does not wait.
func (r *Reconciler) runStopTasks(gs *v1alpha1.GameServer, pod *corev1.Pod) time.Duration {
	if r.Events == nil || gs.Status.Phase != v1alpha1.PhaseRunning ||
		gs.Annotations[gameserver.AnnotationStopSent] == string(pod.UID) {
		return 0
	}
	if gs.Status.StopTasksStartedAt == nil {
		limit := r.Events.StartStopping(gs)
		if limit == 0 {
			return 0
		}
		r.Hub.Daemon(gs.Name, "Running the tasks before stopping...")
		now := metav1.Now()
		gs.Status.StopTasksStartedAt = &now
		stopTaskLimits.Store(string(pod.UID), limit)
		r.setPhase(gs, v1alpha1.PhaseRunning, "running the tasks before stopping")
		return 2 * time.Second
	}
	limit, ok := stopTaskLimits.Load(string(pod.UID))
	if ok && r.Events.Stopping(gs) && time.Since(gs.Status.StopTasksStartedAt.Time) < limit.(time.Duration) {
		r.setPhase(gs, v1alpha1.PhaseRunning, "running the tasks before stopping")
		return 2 * time.Second
	}
	stopTaskLimits.Delete(string(pod.UID))
	return 0
}

// stopGamePod sends the egg's stop command once and kills the pod after the stop timeout.
func (r *Reconciler) stopGamePod(
	ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg, pod *corev1.Pod,
) (reconcile.Result, error) {
	timeout := time.Duration(max(gs.Spec.StopTimeoutSeconds, 1)) * time.Second
	if gs.Status.StopRequestedAt == nil {
		if wait := r.runStopTasks(gs, pod); wait > 0 {
			return reconcile.Result{RequeueAfter: wait}, nil
		}
		r.Hub.Daemon(gs.Name, "Server marked as stopping...")
		if gs.Annotations[gameserver.AnnotationStopSent] != string(pod.UID) {
			if err := r.sendStop(ctx, gs, egg, pod); err != nil {
				r.Hub.Daemon(gs.Name, "Failed to send stop signal: "+err.Error())
			}
		}
		now := metav1.Now()
		gs.Status.StopRequestedAt = &now
		r.setPhase(gs, v1alpha1.PhaseStopping, "")
		return reconcile.Result{RequeueAfter: timeout}, nil
	}
	if elapsed := time.Since(gs.Status.StopRequestedAt.Time); elapsed < timeout {
		r.setPhase(gs, v1alpha1.PhaseStopping, "")
		return reconcile.Result{RequeueAfter: timeout - elapsed}, nil
	}
	r.Hub.Daemon(gs.Name, "Server did not stop in time, terminating process...")
	if err := r.Delete(ctx, pod, client.GracePeriodSeconds(0)); client.IgnoreNotFound(err) != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: 2 * time.Second}, nil
}

// sendStop stops the process the way the egg defines it:
// "^C" / "^SIGINT" -> SIGINT through the TTY, "^SIGTERM"/"^SIGABRT" -> graceful pod
// deletion, any other signal (e.g. "^^C") -> SIGKILL, otherwise a console command.
func (r *Reconciler) sendStop(ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg, pod *corev1.Pod) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	stop := egg.Spec.Stop
	graceful := func() error {
		return client.IgnoreNotFound(r.Delete(ctx, pod, client.GracePeriodSeconds(max(gs.Spec.StopTimeoutSeconds, 1))))
	}
	switch {
	case stop == "":
		return graceful()
	case strings.HasPrefix(stop, "^"):
		switch strings.ToUpper(stop[1:]) {
		case "C", "SIGINT":
			return r.Kube.AttachWrite(ctx, gs.Namespace, pod.Name, gameserver.ContainerName, []byte{0x03})
		case "SIGTERM", "SIGABRT":
			return graceful()
		default:
			return client.IgnoreNotFound(r.Delete(ctx, pod, client.GracePeriodSeconds(0)))
		}
	default:
		return r.Kube.AttachWrite(ctx, gs.Namespace, pod.Name, gameserver.ContainerName, []byte(stop+"\n"))
	}
}
