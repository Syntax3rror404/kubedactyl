package controller

import (
	"context"
	"fmt"
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
	"app/internal/gameserver"
	"app/internal/kube"
)

// How often a storage migration is checked while it waits or copies (the pod events come in between),
// and how long the copy pod may wait for its volume and node before the reason shows.
const (
	migratePoll       = 3 * time.Second
	migrateWaitNormal = 30 * time.Second
)

// reconcileMigration moves the server files to a volume of the storage class in the spec. A
// running server is stopped first (reconcileGame does that while busy is false). Then the files
// and game pods go, a pod copies the files to a new claim and checks the copy, and the data claim
// is swapped for one bound to the new volume. The old volume is deleted when that claim is bound.
// busy is true while the server must wait for the migration.
func (r *Reconciler) reconcileMigration(
	ctx context.Context, gs *v1alpha1.GameServer,
) (res reconcile.Result, busy bool, err error) {
	m := gs.Status.Migration
	switch {
	case gameserver.Migrating(gs) && gs.Status.PodUID == "":
		res, err = r.migrate(ctx, gs)
		return res, true, err
	case gameserver.Migrating(gs):
		return reconcile.Result{}, false, nil
	case m != nil && m.Step != v1alpha1.MigrationFailed:
		res, err = r.finishMigration(ctx, gs, m)
		return res, gs.Status.Migration != nil, err
	}
	return reconcile.Result{}, false, nil
}

// migrate runs the steps up to the swap of the claims.
func (r *Reconciler) migrate(ctx context.Context, gs *v1alpha1.GameServer) (reconcile.Result, error) {
	m := gs.Status.Migration
	if m == nil || m.To != gs.Spec.StorageClass || m.Step == v1alpha1.MigrationFailed {
		m = &v1alpha1.StorageMigration{
			From: gs.Status.StorageClass, To: gs.Spec.StorageClass,
			Step: v1alpha1.MigrationPreparing, StartedAt: metav1.Now(),
		}
		gs.Status.Migration = m
		r.Hub.Daemon(gs.Name, fmt.Sprintf("Moving the server files to storage class %s...", m.To))
	}
	r.setPhase(gs, v1alpha1.PhaseOffline, "")
	if m.Step == v1alpha1.MigrationSwitching {
		return r.switchVolume(ctx, gs, m)
	}
	// The copy must see the files as they are: no game or files pod may write them.
	for _, name := range []string{gameserver.GamePodName(gs.Name), gameserver.FilesPodName(gs.Name)} {
		if gone, err := r.removePod(ctx, gs.Namespace, name); err != nil || !gone {
			return reconcile.Result{RequeueAfter: time.Second}, err
		}
	}
	if err := r.createOwned(ctx, gs, gameserver.MigrateClaim(gs, r.Opts)); err != nil {
		return reconcile.Result{}, err
	}
	pod := &corev1.Pod{}
	err := r.Reader.Get(ctx, migrateKey(gs), pod)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{RequeueAfter: migratePoll}, r.createOwned(ctx, gs, gameserver.MigratePod(gs, r.Opts))
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	return r.followMigrate(ctx, gs, m, pod)
}

// followMigrate records the progress of the copy and moves on when it ended.
func (r *Reconciler) followMigrate(
	ctx context.Context, gs *v1alpha1.GameServer, m *v1alpha1.StorageMigration, pod *corev1.Pod,
) (reconcile.Result, error) {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		return r.startSwitch(ctx, gs, m)
	case corev1.PodFailed:
		_, _, problem := r.migrateLog(ctx, pod)
		if problem == "" {
			_, code, reason := terminated(pod)
			problem = fmt.Sprintf("the copy ended with exit code %d %s", code, reason)
		}
		return reconcile.Result{}, r.failMigration(ctx, gs, m, problem)
	case corev1.PodRunning:
		m.Step = v1alpha1.MigrationCopying
		if done, total, _ := r.migrateLog(ctx, pod); total > 0 {
			m.Done, m.Total = done, total
		}
	default:
		// Waiting for the new volume or a node. The first seconds the scheduler always reports the
		// unbound claim; a problem shows when the wait takes longer.
		if time.Since(pod.CreationTimestamp.Time) > migrateWaitNormal {
			r.setPhase(gs, v1alpha1.PhaseOffline, waitingProblem(pod))
		}
	}
	return reconcile.Result{RequeueAfter: migratePoll}, nil
}

// startSwitch keeps both volumes and points the data claim at the new one. The step is saved
// before any claim is deleted, so a restart of the panel continues with the swap.
func (r *Reconciler) startSwitch(
	ctx context.Context, gs *v1alpha1.GameServer, m *v1alpha1.StorageMigration,
) (reconcile.Result, error) {
	data, err := r.boundClaim(ctx, gs.Namespace, gameserver.PVCName(gs.Name))
	if err != nil {
		return reconcile.Result{}, err
	}
	target, err := r.boundClaim(ctx, gs.Namespace, gameserver.MigrateName(gs.Name))
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, pv := range []string{data.Spec.VolumeName, target.Spec.VolumeName} {
		if err := r.reclaim(ctx, pv, corev1.PersistentVolumeReclaimRetain); err != nil {
			return reconcile.Result{}, err
		}
	}
	patch := client.MergeFrom(gs.DeepCopy())
	if gs.Annotations == nil {
		gs.Annotations = map[string]string{}
	}
	gs.Annotations[gameserver.AnnotationMovedVolume] = target.Spec.VolumeName
	status := gs.Status // Patch overwrites the status with the stored one
	if err := r.Patch(ctx, gs, patch); err != nil {
		return reconcile.Result{}, err
	}
	gs.Status = status
	m.Step, m.Done = v1alpha1.MigrationSwitching, m.Total
	m.Volume, m.PreviousVolume = target.Spec.VolumeName, data.Spec.VolumeName
	r.Hub.Daemon(gs.Name, "Files copied and checked, moving the server to the new volume...")
	return reconcile.Result{RequeueAfter: time.Second}, nil
}

// switchVolume removes the copy pod, the new claim and the old data claim; ensurePVC then creates
// the data claim bound to the new volume, which ends the migration (finishMigration).
func (r *Reconciler) switchVolume(
	ctx context.Context, gs *v1alpha1.GameServer, m *v1alpha1.StorageMigration,
) (reconcile.Result, error) {
	if _, err := r.removePod(ctx, gs.Namespace, gameserver.MigrateName(gs.Name)); err != nil {
		return reconcile.Result{}, err
	}
	if _, err := r.removeClaim(ctx, gs.Namespace, gameserver.MigrateName(gs.Name), ""); err != nil {
		return reconcile.Result{}, err
	}
	_, err := r.removeClaim(ctx, gs.Namespace, gameserver.PVCName(gs.Name), m.Volume)
	return reconcile.Result{RequeueAfter: time.Second}, err
}

// finishMigration binds the new volume to the data claim and deletes the old volume once that
// claim is bound. A migration that was called off before the swap (the spec names the old class
// again) only removes its copy.
func (r *Reconciler) finishMigration(
	ctx context.Context, gs *v1alpha1.GameServer, m *v1alpha1.StorageMigration,
) (reconcile.Result, error) {
	if m.Step != v1alpha1.MigrationSwitching {
		gs.Status.Migration = nil
		r.Hub.Daemon(gs.Name, "Storage migration cancelled.")
		return reconcile.Result{}, r.removeMigration(ctx, gs)
	}
	// The new volume is released for the data claim when the claim of the copy is gone.
	if gone, err := r.removeClaim(ctx, gs.Namespace, gameserver.MigrateName(gs.Name), ""); err != nil || !gone {
		return reconcile.Result{RequeueAfter: time.Second}, err
	}
	err := kube.ReserveVolume(ctx, r.Reader, r.Client, m.Volume, gs.Namespace, gameserver.PVCName(gs.Name))
	if err != nil {
		return reconcile.Result{}, err
	}
	data := &corev1.PersistentVolumeClaim{}
	err = r.Reader.Get(ctx, types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.PVCName(gs.Name)}, data)
	if err != nil || data.Status.Phase != corev1.ClaimBound || data.Spec.VolumeName != m.Volume {
		return reconcile.Result{RequeueAfter: time.Second}, client.IgnoreNotFound(err)
	}
	if err := r.reclaim(ctx, m.PreviousVolume, corev1.PersistentVolumeReclaimDelete); err != nil {
		return reconcile.Result{}, err
	}
	gs.Status.Migration = nil
	r.Hub.Daemon(gs.Name, fmt.Sprintf("Storage migration to %s finished.", m.To))
	return reconcile.Result{}, nil
}

// failMigration removes the copy and keeps the server on its volume (the spec names the old
// class again); the error stays in the status until the next migration.
func (r *Reconciler) failMigration(
	ctx context.Context, gs *v1alpha1.GameServer, m *v1alpha1.StorageMigration, problem string,
) error {
	if err := r.removeMigration(ctx, gs); err != nil {
		return err
	}
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.StorageClass = m.From
	status := gs.Status
	if err := r.Patch(ctx, gs, patch); err != nil {
		return err
	}
	gs.Status = status
	m.Step, m.Error = v1alpha1.MigrationFailed, problem
	r.Hub.Daemon(gs.Name, "Storage migration failed: "+problem)
	return nil
}

// removeMigration deletes the copy pod and the new claim together with its volume.
func (r *Reconciler) removeMigration(ctx context.Context, gs *v1alpha1.GameServer) error {
	if _, err := r.removePod(ctx, gs.Namespace, gameserver.MigrateName(gs.Name)); err != nil {
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Reader.Get(ctx, migrateKey(gs), pvc); err != nil {
		return client.IgnoreNotFound(err)
	}
	if err := r.reclaim(ctx, pvc.Spec.VolumeName, corev1.PersistentVolumeReclaimDelete); err != nil {
		return err
	}
	return client.IgnoreNotFound(r.Delete(ctx, pvc))
}

// migrateLog reads the last lines of the copy pod (see gameserver.MigrateLog); a failed read
// reports nothing.
func (r *Reconciler) migrateLog(ctx context.Context, pod *corev1.Pod) (done, total int64, problem string) {
	raw, err := r.Kube.Clientset.CoreV1().Pods(pod.Namespace).
		GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: ptr.To[int64](20)}).DoRaw(ctx)
	if err != nil {
		return 0, 0, ""
	}
	return gameserver.MigrateLog(string(raw))
}

// removePod deletes a pod at once; gone is true when it does not exist any more.
func (r *Reconciler) removePod(ctx context.Context, namespace, name string) (gone bool, err error) {
	pod := &corev1.Pod{}
	err = r.Reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, pod)
	if apierrors.IsNotFound(err) {
		return true, nil
	}
	if err != nil || pod.DeletionTimestamp != nil {
		return false, err
	}
	return false, client.IgnoreNotFound(r.Delete(ctx, pod, client.GracePeriodSeconds(0)))
}

// removeClaim deletes a claim unless it is bound to the volume keep; gone is true when it does not
// exist (any more) or is kept.
func (r *Reconciler) removeClaim(ctx context.Context, namespace, name, keep string) (gone bool, err error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err = r.Reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, pvc)
	switch {
	case apierrors.IsNotFound(err):
		return true, nil
	case err != nil:
		return false, err
	case keep != "" && pvc.Spec.VolumeName == keep:
		return true, nil
	case pvc.DeletionTimestamp != nil:
		return false, nil
	}
	return false, client.IgnoreNotFound(r.Delete(ctx, pvc))
}

// reclaim sets what happens to a volume when its claim is gone (see kube.SetReclaimPolicy).
func (r *Reconciler) reclaim(ctx context.Context, name string, policy corev1.PersistentVolumeReclaimPolicy) error {
	return kube.SetReclaimPolicy(ctx, r.Reader, r.Client, name, policy)
}

// createOwned creates an object of the server unless it exists.
func (r *Reconciler) createOwned(ctx context.Context, gs *v1alpha1.GameServer, obj client.Object) error {
	if err := controllerutil.SetControllerReference(gs, obj, r.Scheme()); err != nil {
		return err
	}
	return client.IgnoreAlreadyExists(r.Create(ctx, obj))
}

// boundClaim reads a claim that must be bound to a volume.
func (r *Reconciler) boundClaim(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, pvc); err != nil {
		return nil, err
	}
	if pvc.Spec.VolumeName == "" || pvc.Status.Phase != corev1.ClaimBound {
		return nil, fmt.Errorf("claim %s is not bound", name)
	}
	return pvc, nil
}

func migrateKey(gs *v1alpha1.GameServer) types.NamespacedName {
	return types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.MigrateName(gs.Name)}
}
