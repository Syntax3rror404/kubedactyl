package controller

import (
	"context"
	"fmt"
	"strconv"
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
	"app/internal/gameserver"
	"app/internal/kube"
)

// reconcileInstall runs the egg install script for a new install revision: it stops the game
// server, follows the install pod of the current revision and starts one when none exists.
func (r *Reconciler) reconcileInstall(
	ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg,
) (reconcile.Result, error) {
	if res, done, err := r.stopGameForInstall(ctx, gs); done {
		return res, err
	}
	pod := &corev1.Pod{}
	key := types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.InstallPodName(gs.Name)}
	err := r.Reader.Get(ctx, key, pod)
	revision := strconv.FormatInt(gs.Spec.InstallRevision, 10)
	switch {
	case err == nil && pod.Annotations[gameserver.AnnotationInstallRevision] == revision:
		return r.followInstall(gs, pod)
	case err == nil:
		return r.removeOldInstall(ctx, gs, pod)
	case !apierrors.IsNotFound(err):
		return reconcile.Result{}, err
	}
	return r.startInstall(ctx, gs, egg)
}

// stopGameForInstall stops a running game pod first (killed after a 10 second timeout);
// done is true while a game pod still exists.
func (r *Reconciler) stopGameForInstall(
	ctx context.Context,
	gs *v1alpha1.GameServer,
) (res reconcile.Result, done bool, err error) {
	game := &corev1.Pod{}
	err = r.Reader.Get(ctx, types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.GamePodName(gs.Name)}, game)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, false, nil
	}
	if err != nil {
		return reconcile.Result{}, true, err
	}
	if game.DeletionTimestamp == nil {
		grace := int64(10)
		if game.Annotations[gameserver.AnnotationExitHandled] != "" {
			grace = 0 // already exited
		} else {
			r.Hub.Daemon(gs.Name, "Stopping server for reinstallation...")
		}
		if err := r.Delete(ctx, game, client.GracePeriodSeconds(grace)); client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, true, err
		}
	}
	r.setPhase(gs, v1alpha1.PhaseStopping, "stopping server for reinstallation")
	return reconcile.Result{RequeueAfter: 2 * time.Second}, true, nil
}

// followInstall streams the output of the install pod and records its result.
func (r *Reconciler) followInstall(gs *v1alpha1.GameServer, pod *corev1.Pod) (reconcile.Result, error) {
	if pod.DeletionTimestamp != nil {
		return reconcile.Result{RequeueAfter: 2 * time.Second}, nil
	}
	r.Hub.Follow(gs.Name, pod, console.KindInstall, nil)
	if done, code, _ := terminated(pod); done {
		// The exit code of install scripts is ignored (many egg scripts end with an error);
		// it is recorded for the UI.
		gs.Status.InstalledRevision = gs.Spec.InstallRevision
		gs.Status.InstallExitCode = ptr.To(code)
		r.setPhase(gs, v1alpha1.PhaseOffline, "")
		r.Hub.Daemon(gs.Name, fmt.Sprintf("Installation process completed (exit code %d).", code))
		// Give the log stream a moment to deliver the last lines.
		go func(ns, server string) {
			time.Sleep(3 * time.Second)
			r.Trigger(ns, server)
		}(gs.Namespace, gs.Name)
		return reconcile.Result{}, nil
	}
	if problem := waitingProblem(pod); problem != "" {
		r.setPhase(gs, v1alpha1.PhaseInstallFailed, problem)
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}
	r.setPhase(gs, v1alpha1.PhaseInstalling, "")
	return reconcile.Result{}, nil
}

// removeOldInstall deletes the install pod of an older revision.
func (r *Reconciler) removeOldInstall(
	ctx context.Context, gs *v1alpha1.GameServer, pod *corev1.Pod,
) (reconcile.Result, error) {
	if pod.DeletionTimestamp == nil {
		if err := r.Delete(ctx, pod, client.GracePeriodSeconds(0)); client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}
	}
	r.setPhase(gs, v1alpha1.PhaseInstalling, "cleaning up previous installation")
	return reconcile.Result{RequeueAfter: 2 * time.Second}, nil
}

// startInstall creates the script ConfigMap and the install pod once the volume is bound.
func (r *Reconciler) startInstall(
	ctx context.Context, gs *v1alpha1.GameServer, egg *v1alpha1.Egg,
) (reconcile.Result, error) {
	// A fresh claim takes a few seconds to be provisioned.
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Reader.Get(
		ctx, types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.PVCName(gs.Name)}, pvc,
	); err != nil ||
		pvc.Status.Phase != corev1.ClaimBound {
		r.setPhase(gs, v1alpha1.PhaseInstalling, "waiting for the volume to be provisioned")
		return reconcile.Result{RequeueAfter: 2 * time.Second}, client.IgnoreNotFound(err)
	}
	cm := gameserver.InstallConfigMap(gs, egg)
	cur := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: cm.Name, Namespace: cm.Namespace}}
	if err := kube.CreateOrPatch(ctx, r.Reader, r.Client, cur, func() error {
		cur.Labels = cm.Labels
		cur.Data = cm.Data
		return controllerutil.SetControllerReference(gs, cur, r.Scheme())
	}); err != nil {
		return reconcile.Result{}, err
	}
	want := gameserver.InstallPod(gs, egg, r.Opts)
	if err := controllerutil.SetControllerReference(gs, want, r.Scheme()); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.Create(ctx, want); client.IgnoreAlreadyExists(err) != nil {
		return reconcile.Result{}, err
	}
	r.Hub.Daemon(gs.Name, "Starting installation process, this could take a few minutes...")
	gs.Status.InstallExitCode = nil
	r.setPhase(gs, v1alpha1.PhaseInstalling, "")
	return reconcile.Result{}, nil
}
