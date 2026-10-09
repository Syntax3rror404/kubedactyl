// Package controller reconciles GameServer resources into pods, volumes and services.
package controller

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"app/api/v1alpha1"
	"app/internal/console"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
	"app/internal/settings"
	"app/internal/tenancy"
)

// ServerEvents runs the "Tasks" of a server: after it was marked as running and before it is
// stopped. The stop waits while Stopping reports true, at most the returned duration.
type ServerEvents interface {
	Started(gs *v1alpha1.GameServer)
	StartStopping(gs *v1alpha1.GameServer) time.Duration
	Stopping(gs *v1alpha1.GameServer) bool
}

// Reconciler reconciles GameServer objects.
type Reconciler struct {
	client.Client
	// Reader reads directly from the API server (no cache): for the server itself, its pods,
	// volume claims and everything else whose current state decides what happens next.
	Reader client.Reader
	Kube   *kube.Client
	Hub    *console.Hub
	Opts   gameserver.Options
	// Files records file manager use; the files pod only exists while it is used.
	Files *files.Activity
	// Pools resolves load balancer pool names to service labels.
	Pools *settings.PoolResolver
	// Events runs the tasks of a server at its start and before its stop (schedule.Runner).
	Events ServerEvents
	Log    *slog.Logger

	events chan event.GenericEvent
	// seenTerminated records when a finished pod was first seen (see removeTerminated).
	seenTerminated sync.Map
}

// Trigger requests a reconcile of a server, e.g. after its done line was printed.
func (r *Reconciler) Trigger(namespace, server string) {
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: server, Namespace: namespace}}
	select {
	case r.events <- event.GenericEvent{Object: gs}:
	default:
	}
}

// SetupWithManager registers the controller.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.events = make(chan event.GenericEvent, 1024)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GameServer{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		WatchesRawSource(source.Channel(r.events, &handler.EnqueueRequestForObject{})).
		// An edited egg can change what a running server would start with ("restart required").
		// Only spec changes count: the status of an egg (its update checks) does not affect servers.
		Watches(&v1alpha1.Egg{}, handler.EnqueueRequestsFromMapFunc(r.serversOfEgg),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Different servers reconcile in parallel; the same server never does.
		WithOptions(crcontroller.Options{MaxConcurrentReconciles: 4}).
		Complete(r)
}

// serversOfEgg maps an egg of this installation to the servers created from it.
func (r *Reconciler) serversOfEgg(ctx context.Context, obj client.Object) []reconcile.Request {
	if obj.GetNamespace() != r.Opts.Namespace {
		return nil
	}
	var list v1alpha1.GameServerList
	if err := r.List(ctx, &list); err != nil {
		return nil
	}
	var out []reconcile.Request
	for _, gs := range list.Items {
		if gs.Spec.EggRef == obj.GetName() && tenancy.Owns(gs.Namespace) {
			out = append(out, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&gs)})
		}
	}
	return out
}

// Reconcile implements reconcile.Reconciler.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (result reconcile.Result, err error) {
	defer r.logReconcile(ctx, req.Name, time.Now(), &result, &err)
	if !tenancy.Owns(req.Namespace) {
		return reconcile.Result{}, nil // belongs to another installation
	}
	// Read the server straight from the API: a power change and the pod event it causes
	// can arrive before the cache has the new spec (e.g. kill looked like a crash).
	gs := &v1alpha1.GameServer{}
	if err := r.Reader.Get(ctx, req.NamespacedName, gs); err != nil {
		if apierrors.IsNotFound(err) {
			r.Hub.Remove(req.Name)
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	if !gs.DeletionTimestamp.IsZero() {
		return r.finalize(ctx, gs)
	}
	if controllerutil.AddFinalizer(gs, gameserver.Finalizer) {
		if err := r.Update(ctx, gs); err != nil {
			return reconcile.Result{}, err
		}
	}

	before := gs.Status.DeepCopy()
	result, err = r.reconcile(ctx, gs)
	if err != nil {
		gs.Status.Message = err.Error()
	}
	gs.Status.ObservedGeneration = gs.Generation
	// Safety net against missed watch events: re-check active servers regularly.
	if err == nil && result.RequeueAfter == 0 && isActive(gs.Status.Phase) {
		result.RequeueAfter = 20 * time.Second
	}
	if equality.Semantic.DeepEqual(before, &gs.Status) {
		return result, err
	}
	if uerr := r.Status().Update(ctx, gs); uerr != nil {
		if apierrors.IsConflict(uerr) {
			return reconcile.Result{RequeueAfter: time.Second}, nil
		}
		return reconcile.Result{}, uerr
	}
	if before.Phase != gs.Status.Phase {
		r.Hub.Status(gs.Name, string(gs.Status.Phase))
	}
	return result, err
}

func isActive(phase v1alpha1.Phase) bool {
	switch phase {
	case v1alpha1.PhaseRunning, v1alpha1.PhaseStarting, v1alpha1.PhaseStopping, v1alpha1.PhaseInstalling:
		return true
	}
	return false
}

// logReconcile logs every reconcile (debug; warning when it took longer than 5 seconds).
func (r *Reconciler) logReconcile(
	ctx context.Context, server string, start time.Time, result *reconcile.Result, err *error,
) {
	d := time.Since(start)
	level := slog.LevelDebug
	if d > 5*time.Second {
		level = slog.LevelWarn
	}
	r.Log.Log(
		ctx, level, "reconciled", "server", server, "duration", d.Round(time.Millisecond), "requeueAfter",
		result.RequeueAfter, "err", *err,
	)
}

func (r *Reconciler) setPhase(gs *v1alpha1.GameServer, phase v1alpha1.Phase, msg string) {
	gs.Status.Phase = phase
	gs.Status.Message = msg
}

func (r *Reconciler) reconcile(ctx context.Context, gs *v1alpha1.GameServer) (reconcile.Result, error) {
	egg := &v1alpha1.Egg{}
	// Eggs are shared by all users and live in the panel namespace.
	if err := r.Get(ctx, types.NamespacedName{Namespace: r.Opts.Namespace, Name: gs.Spec.EggRef}, egg); err != nil {
		if apierrors.IsNotFound(err) {
			r.setPhase(gs, v1alpha1.PhasePending, fmt.Sprintf("egg %q not found", gs.Spec.EggRef))
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return reconcile.Result{}, err
	}
	if err := r.ensurePVC(ctx, gs); err != nil {
		return reconcile.Result{}, fmt.Errorf("volume: %w", err)
	}
	if err := r.ensureService(ctx, gs); err != nil {
		return reconcile.Result{}, fmt.Errorf("service: %w", err)
	}
	if res, busy, err := r.reconcileMigration(ctx, gs); busy {
		return res, err
	}
	// The files pod is created on demand: for the file manager and the pre-start steps.
	if r.Files.Active(files.RefOf(gs)) {
		if _, err := r.ensureFilesPod(ctx, gs); err != nil {
			return reconcile.Result{}, fmt.Errorf("files pod: %w", err)
		}
	}

	// A server transferred from another owner starts with an empty status but is installed.
	if rev, _ := strconv.ParseInt(
		gs.Annotations[gameserver.AnnotationInstalledRevision], 10, 64,
	); gs.Status.InstalledRevision < rev {
		gs.Status.InstalledRevision = rev
	}
	if gs.Status.InstalledRevision < gs.Spec.InstallRevision {
		if gs.Spec.SkipInstall {
			gs.Status.InstalledRevision = gs.Spec.InstallRevision
		} else {
			return r.reconcileInstall(ctx, gs, egg)
		}
	}
	cleanup, err := r.cleanupInstall(ctx, gs)
	if err != nil {
		return reconcile.Result{}, err
	}
	result, err := r.reconcileGame(ctx, gs, egg)
	if cleanup > 0 && (result.RequeueAfter == 0 || cleanup < result.RequeueAfter) {
		result.RequeueAfter = cleanup
	}
	return result, err
}

// finalize releases the volumes of the server: the "longhorn" storage class retains volumes,
// so the reclaim policy is switched to Delete before the claims go away, unless the server was
// transferred to another owner, which keeps using the volume. A storage migration can hold a
// second claim and, while it switches, two released volumes.
func (r *Reconciler) finalize(ctx context.Context, gs *v1alpha1.GameServer) (reconcile.Result, error) {
	if !controllerutil.ContainsFinalizer(gs, gameserver.Finalizer) {
		return reconcile.Result{}, nil
	}
	r.Hub.Remove(gs.Name)
	if gs.Annotations[gameserver.AnnotationKeepVolume] == "" {
		var volumes []string
		if m := gs.Status.Migration; m != nil {
			volumes = append(volumes, m.Volume, m.PreviousVolume)
		}
		for _, name := range []string{gameserver.PVCName(gs.Name), gameserver.MigrateName(gs.Name)} {
			pvc := &corev1.PersistentVolumeClaim{}
			err := r.Reader.Get(ctx, types.NamespacedName{Namespace: gs.Namespace, Name: name}, pvc)
			if client.IgnoreNotFound(err) != nil {
				return reconcile.Result{}, err
			}
			volumes = append(volumes, pvc.Spec.VolumeName)
		}
		for _, pv := range volumes {
			if err := r.reclaim(ctx, pv, corev1.PersistentVolumeReclaimDelete); err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	controllerutil.RemoveFinalizer(gs, gameserver.Finalizer)
	return reconcile.Result{}, r.Update(ctx, gs)
}
