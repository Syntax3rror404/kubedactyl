package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetReclaimPolicy changes what happens to a persistent volume when its claim is gone: Retain
// keeps it for another claim, Delete removes it (at once when it is released already). A missing
// volume (or none named) is no error.
func SetReclaimPolicy(
	ctx context.Context, reader client.Reader, c client.Client,
	name string, policy corev1.PersistentVolumeReclaimPolicy,
) error {
	if name == "" {
		return nil
	}
	pv := &corev1.PersistentVolume{}
	if err := reader.Get(ctx, client.ObjectKey{Name: name}, pv); err != nil {
		return client.IgnoreNotFound(err)
	}
	if pv.Spec.PersistentVolumeReclaimPolicy == policy {
		return nil
	}
	patch := client.MergeFrom(pv.DeepCopy())
	pv.Spec.PersistentVolumeReclaimPolicy = policy
	return c.Patch(ctx, pv, patch)
}

// ReserveVolume binds a released persistent volume to the claim namespace/name, which may not exist
// yet (no UID): Kubernetes binds them as soon as the claim asks for this volume.
func ReserveVolume(ctx context.Context, reader client.Reader, c client.Client, name, namespace, claim string) error {
	pv := &corev1.PersistentVolume{}
	if err := reader.Get(ctx, client.ObjectKey{Name: name}, pv); err != nil {
		return err
	}
	if ref := pv.Spec.ClaimRef; ref != nil && ref.Namespace == namespace && ref.Name == claim {
		return nil
	}
	patch := client.MergeFrom(pv.DeepCopy())
	pv.Spec.ClaimRef = &corev1.ObjectReference{
		Kind: "PersistentVolumeClaim", APIVersion: "v1", Namespace: namespace, Name: claim,
	}
	return c.Patch(ctx, pv, patch)
}
