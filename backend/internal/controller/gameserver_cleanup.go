package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
)

// Terminated game and install pods are deleted so they do not pile up in the cluster.
// terminatedGrace lets the log stream deliver the last lines to open consoles first;
// the stream ends by itself when the container is gone.
const terminatedGrace = 3 * time.Second

// removeTerminated deletes a finished pod shortly after it ended. It returns how long to
// wait before calling again (0 when the pod is gone or being deleted).
func (r *Reconciler) removeTerminated(ctx context.Context, pod *corev1.Pod) (time.Duration, error) {
	if pod.DeletionTimestamp != nil {
		return 0, nil
	}
	first, _ := r.seenTerminated.LoadOrStore(pod.UID, time.Now())
	if wait := terminatedGrace - time.Since(first.(time.Time)); wait > 0 {
		return wait, nil
	}
	r.seenTerminated.Delete(pod.UID)
	return 0, client.IgnoreNotFound(r.Delete(ctx, pod, client.GracePeriodSeconds(0)))
}

// cleanupInstall removes the finished install pod and its script once the installation
// is recorded. It returns how long to wait before the pod can go.
func (r *Reconciler) cleanupInstall(ctx context.Context, gs *v1alpha1.GameServer) (time.Duration, error) {
	pod := &corev1.Pod{}
	err := r.Reader.Get(
		ctx, types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.InstallPodName(gs.Name)}, pod,
	)
	if apierrors.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if done, _, _ := terminated(pod); !done {
		return 0, nil
	}
	wait, err := r.removeTerminated(ctx, pod)
	if err != nil || wait > 0 {
		return wait, err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: gameserver.InstallConfigName(gs.Name), Namespace: gs.Namespace},
	}
	return 0, client.IgnoreNotFound(r.Delete(ctx, cm))
}
