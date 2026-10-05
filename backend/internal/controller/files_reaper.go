package controller

import (
	"context"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
	"app/internal/tenancy"
)

// FilesReaper removes files pods that were not used for files.IdleTimeout, so idle
// servers do not keep a pod running. Before a pod goes, the disk usage is measured and
// stored in the server status. It runs as a manager runnable.
type FilesReaper struct {
	Client client.Client
	// Reader lists the files pods uncached: a lagging informer would still show a removed
	// pod and the reaper would delete its successor of the same name.
	Reader client.Reader
	Kube   *kube.Client
	Files  *files.Activity
	Log    *slog.Logger
}

// reapInterval is how often the reaper looks for idle files pods.
const reapInterval = 10 * time.Second

// Start implements manager.Runnable.
func (f *FilesReaper) Start(ctx context.Context) error {
	t := time.NewTicker(reapInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := f.reap(ctx); err != nil {
				f.Log.Warn("removing idle files pods", "err", err)
			}
		}
	}
}

func (f *FilesReaper) reap(ctx context.Context) error {
	var pods corev1.PodList
	if err := f.Reader.List(ctx, &pods, client.MatchingLabels{gameserver.LabelRole: gameserver.RoleFiles}); err != nil {
		return err
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		ref := files.Ref{Namespace: pod.Namespace, Name: pod.Labels[gameserver.LabelServer]}
		if !tenancy.Owns(pod.Namespace) || ref.Name == "" || !idlePod(pod, f.Files.Last(ref), time.Now()) {
			continue
		}
		if pod.Status.Phase == corev1.PodRunning {
			f.recordDiskUsage(ctx, ref, pod.Name)
		}
		// Only this pod: never a newer one with the same name.
		err := f.Client.Delete(ctx, pod, client.Preconditions{UID: &pod.UID})
		if apierrors.IsConflict(err) {
			continue
		}
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		f.Files.Forget(ref)
		f.Log.Debug("removed idle files pod", "server", ref.Name)
	}
	return nil
}

// filesStartLimit is how long a files pod may take to become ready before it is removed
// anyway. Attaching a volume that another pod just released can take over a minute.
const filesStartLimit = 10 * time.Minute

// idlePod reports whether a files pod can go: its time is up (files.StopsAt: not used for
// files.IdleTimeout since it became ready), or it has been starting for longer than
// filesStartLimit.
func idlePod(pod *corev1.Pod, lastUse, now time.Time) bool {
	if pod.DeletionTimestamp != nil {
		return false
	}
	readySince := files.ReadySince(pod)
	if readySince.IsZero() {
		return now.Sub(pod.CreationTimestamp.Time) > filesStartLimit
	}
	return !now.Before(files.StopsAt(lastUse, readySince))
}

// recordDiskUsage stores the size of the server files in the status (best effort).
func (f *FilesReaper) recordDiskUsage(ctx context.Context, ref files.Ref, pod string) {
	bytes, err := gameserver.DiskUsage(ctx, f.Kube, ref.Namespace, pod, gameserver.FilesContainerName)
	if err != nil {
		f.Log.Debug("measuring disk usage", "server", ref.Name, "err", err)
		return
	}
	gs := &v1alpha1.GameServer{}
	gs.Namespace, gs.Name = ref.Namespace, ref.Name
	patch := client.MergeFrom(gs.DeepCopy())
	now := metav1.Now()
	gs.Status.DiskUsedBytes, gs.Status.DiskMeasuredAt = &bytes, &now
	if err := f.Client.Status().Patch(ctx, gs, patch); err != nil {
		f.Log.Debug("storing disk usage", "server", ref.Name, "err", err)
	}
}
