package serverctl

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/tenancy"
)

// Errors of transfers.
var (
	ErrSameOwner = errors.New("the server already belongs to this user")
	ErrFilesBusy = errors.New("a backup, restore or download is running, wait until it has finished")
	ErrNoVolume  = errors.New("the server has no bound data volume yet")
)

// How long a transfer waits for the old server object, its claim and its service to go.
var (
	transferPoll = 500 * time.Millisecond
	transferWait = 2 * time.Minute
)

// Transfer moves a stopped server to the namespace of another user. The data volume moves
// with it: it is retained, the old server object is deleted (its finalizer keeps the volume),
// the volume is bound to a claim of the same name in the new namespace and the server is
// created there with the same spec. The address stays: it is fixed to the current one.
// Returns the new server object.
func (o *Ops) Transfer(
	ctx context.Context, gs *v1alpha1.GameServer, owner, namespace string,
) (*v1alpha1.GameServer, error) {
	if gs.Namespace == namespace {
		return nil, ErrSameOwner
	}
	if err := Stopped(gs); err != nil {
		return nil, err
	}
	if o.Files != nil &&
		o.Files.Busy(
			files.RefOf(gs), files.JobPull, files.JobBackup, files.JobRestore,
		) {
		return nil, ErrFilesBusy
	}
	pv, err := o.retainVolume(ctx, gs)
	if err != nil {
		return nil, err
	}
	moved := transferred(gs, owner, namespace, pv.Name)
	if err := o.removeKeepingVolume(ctx, gs); err != nil {
		return nil, err
	}
	if err := o.rebind(ctx, pv, moved); err != nil {
		return nil, fmt.Errorf("volume %s is kept, binding it failed: %w", pv.Name, err)
	}
	if err := o.Client.Create(ctx, moved); err != nil {
		return nil, fmt.Errorf("volume %s is kept, creating the server failed: %w", pv.Name, err)
	}
	return moved, nil
}

// retainVolume returns the server's persistent volume after making sure it survives its claim.
func (o *Ops) retainVolume(ctx context.Context, gs *v1alpha1.GameServer) (*corev1.PersistentVolume, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := o.Reader.Get(
		ctx, client.ObjectKey{Namespace: gs.Namespace, Name: gameserver.PVCName(gs.Name)}, pvc,
	); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNoVolume
		}
		return nil, err
	}
	if pvc.Spec.VolumeName == "" || pvc.Status.Phase != corev1.ClaimBound {
		return nil, ErrNoVolume
	}
	pv := &corev1.PersistentVolume{}
	if err := o.Reader.Get(ctx, client.ObjectKey{Name: pvc.Spec.VolumeName}, pv); err != nil {
		return nil, err
	}
	if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		patch := client.MergeFrom(pv.DeepCopy())
		pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		if err := o.Client.Patch(ctx, pv, patch); err != nil {
			return nil, err
		}
	}
	return pv, nil
}

// transferred is the server object for the new owner: same name and spec, the load balancer
// address fixed to the current one, and annotations that bind the moved volume and skip the installation.
func transferred(gs *v1alpha1.GameServer, owner, namespace, volume string) *v1alpha1.GameServer {
	moved := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: gs.Name, Namespace: namespace,
			Labels: maps.Clone(gs.Labels),
			Annotations: map[string]string{
				gameserver.AnnotationMovedVolume:       volume,
				gameserver.AnnotationInstalledRevision: strconv.FormatInt(gs.Status.InstalledRevision, 10),
			},
		},
		Spec: *gs.Spec.DeepCopy(),
	}
	if moved.Labels == nil {
		moved.Labels = map[string]string{}
	}
	moved.Labels[tenancy.LabelOwner] = owner
	if moved.Spec.LoadBalancerIP == "" && !gameserver.ClusterOnly(gs) {
		// All addresses: fixing only one would cost the server its IPv6 address.
		moved.Spec.LoadBalancerIP = cmp.Or(strings.Join(gs.Status.Addresses, ","), gs.Status.Address)
	}
	return moved
}

// removeKeepingVolume deletes the old server object without its volume and waits until the
// server, its claim and its service are gone (the name is reused in the new namespace).
func (o *Ops) removeKeepingVolume(ctx context.Context, gs *v1alpha1.GameServer) error {
	patch := client.MergeFrom(gs.DeepCopy())
	if gs.Annotations == nil {
		gs.Annotations = map[string]string{}
	}
	gs.Annotations[gameserver.AnnotationKeepVolume] = "true"
	if err := o.Client.Patch(ctx, gs, patch); err != nil {
		return err
	}
	if err := o.Client.Delete(ctx, gs); client.IgnoreNotFound(err) != nil {
		return err
	}
	deadline := time.Now().Add(transferWait)
	for _, old := range []struct {
		what string
		obj  client.Object
		key  client.ObjectKey
	}{
		{"server", &v1alpha1.GameServer{}, client.ObjectKeyFromObject(gs)},
		{
			"data volume", &corev1.PersistentVolumeClaim{},
			client.ObjectKey{Namespace: gs.Namespace, Name: gameserver.PVCName(gs.Name)},
		},
		{"service", &corev1.Service{}, client.ObjectKeyFromObject(gs)},
	} {
		if err := o.waitGone(ctx, old.key, old.obj, deadline); err != nil {
			return fmt.Errorf("the old %s of %s: %w", old.what, gs.Name, err)
		}
	}
	return nil
}

// waitGone polls until the object does not exist any more.
func (o *Ops) waitGone(ctx context.Context, key client.ObjectKey, obj client.Object, deadline time.Time) error {
	for {
		err := o.Reader.Get(ctx, key, obj)
		switch {
		case apierrors.IsNotFound(err):
			return nil
		case err != nil:
			return err
		case time.Now().After(deadline):
			return errors.New("was not removed in time")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(transferPoll):
		}
	}
}

// rebind reserves the released volume for the claim of the moved server, which the
// controller creates with the same name in the new namespace.
func (o *Ops) rebind(ctx context.Context, pv *corev1.PersistentVolume, moved *v1alpha1.GameServer) error {
	cur := &corev1.PersistentVolume{}
	if err := o.Reader.Get(ctx, client.ObjectKeyFromObject(pv), cur); err != nil {
		return err
	}
	patch := client.MergeFrom(cur.DeepCopy())
	cur.Spec.ClaimRef = &corev1.ObjectReference{
		Kind: "PersistentVolumeClaim", APIVersion: "v1",
		Namespace: moved.Namespace, Name: gameserver.PVCName(moved.Name),
	}
	return o.Client.Patch(ctx, cur, patch)
}
