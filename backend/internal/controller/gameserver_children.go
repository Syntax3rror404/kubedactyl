package controller

import (
	"context"
	"maps"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"app/api/v1alpha1"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
)

func (r *Reconciler) ensurePVC(ctx context.Context, gs *v1alpha1.GameServer) error {
	want := gameserver.PVC(gs, r.Opts)
	cur := &corev1.PersistentVolumeClaim{}
	// Uncached: the cache may not show a claim that was created a moment ago.
	err := r.Reader.Get(ctx, client.ObjectKeyFromObject(want), cur)
	if apierrors.IsNotFound(err) {
		if err := controllerutil.SetControllerReference(gs, want, r.Scheme()); err != nil {
			return err
		}
		return client.IgnoreAlreadyExists(r.Create(ctx, want))
	}
	if err != nil {
		return err
	}
	// Longhorn volumes can be expanded online, never shrunk.
	wantSize := want.Spec.Resources.Requests[corev1.ResourceStorage]
	curSize := cur.Spec.Resources.Requests[corev1.ResourceStorage]
	if wantSize.Cmp(curSize) > 0 {
		patch := client.MergeFrom(cur.DeepCopy())
		cur.Spec.Resources.Requests[corev1.ResourceStorage] = wantSize
		return r.Patch(ctx, cur, patch)
	}
	return nil
}

func (r *Reconciler) ensureService(ctx context.Context, gs *v1alpha1.GameServer) error {
	var poolLabels map[string]string
	if gs.Spec.LoadBalancerPool != "" && !gameserver.ClusterOnly(gs) {
		var err error
		if poolLabels, err = r.Pools.ServiceLabels(ctx, gs.Spec.LoadBalancerPool); err != nil {
			return err
		}
	}
	want := gameserver.Service(gs, poolLabels)
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: want.Name, Namespace: want.Namespace}}
	err := kube.CreateOrPatch(ctx, r.Reader, r.Client, svc, func() error {
		want.Annotations[gameserver.AnnotationFixedIPs] = gameserver.FixedIPs(
			gs.Spec.LoadBalancerIP, svc.Spec.IPFamilies,
		)
		mergeServiceMetadata(svc, want)
		svc.Spec.Type = want.Spec.Type
		svc.Spec.Selector = want.Spec.Selector
		svc.Spec.ExternalTrafficPolicy = want.Spec.ExternalTrafficPolicy
		svc.Spec.IPFamilyPolicy = want.Spec.IPFamilyPolicy
		svc.Spec.Ports = mergePorts(svc.Spec.Ports, want.Spec.Ports, want.Spec.Type)
		return controllerutil.SetControllerReference(gs, svc, r.Scheme())
	})
	if err != nil {
		return err
	}
	gs.Status.Addresses = gameserver.LoadBalancerIPs(svc)
	gs.Status.Address = ""
	if gameserver.ClusterOnly(gs) {
		gs.Status.Addresses = nil
		gs.Status.Address = gameserver.ClusterAddress(gs)
	} else if len(gs.Status.Addresses) > 0 {
		gs.Status.Address = gs.Status.Addresses[0]
	}
	return nil
}

// mergeServiceMetadata sets the labels and annotations the panel manages and keeps foreign
// ones. The selector labels of a previous pool are removed when the server moved (their keys
// are remembered in the pool-labels annotation).
func mergeServiceMetadata(svc, want *corev1.Service) {
	if svc.Labels == nil {
		svc.Labels = map[string]string{}
	}
	for k := range strings.SplitSeq(svc.Annotations[gameserver.AnnotationPoolLabels], ",") {
		if _, keep := want.Labels[k]; !keep {
			delete(svc.Labels, k)
		}
	}
	maps.Copy(svc.Labels, want.Labels)
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	for _, key := range []string{gameserver.AnnotationFixedIPs, gameserver.AnnotationPoolLabels} {
		if v := want.Annotations[key]; v != "" {
			svc.Annotations[key] = v
		} else {
			delete(svc.Annotations, key)
		}
	}
}

// mergePorts keeps the node ports allocated by Kubernetes for unchanged ports; a ClusterIP
// service has none.
func mergePorts(cur, want []corev1.ServicePort, typ corev1.ServiceType) []corev1.ServicePort {
	if typ == corev1.ServiceTypeClusterIP {
		return want
	}
	nodePorts := map[string]int32{}
	for _, p := range cur {
		nodePorts[p.Name] = p.NodePort
	}
	out := make([]corev1.ServicePort, len(want))
	for i, p := range want {
		p.NodePort = nodePorts[p.Name]
		out[i] = p
	}
	return out
}

func (r *Reconciler) ensureFilesPod(ctx context.Context, gs *v1alpha1.GameServer) (bool, error) {
	pod := &corev1.Pod{}
	// Uncached like the game pod: the files pod comes and goes and its state must be exact.
	err := r.Reader.Get(ctx, types.NamespacedName{Namespace: gs.Namespace, Name: gameserver.FilesPodName(gs.Name)}, pod)
	if apierrors.IsNotFound(err) {
		want := gameserver.FilesPod(gs, r.Opts)
		if err := controllerutil.SetControllerReference(gs, want, r.Scheme()); err != nil {
			return false, err
		}
		return false, client.IgnoreAlreadyExists(r.Create(ctx, want))
	}
	if err != nil {
		return false, err
	}
	if !metav1.IsControlledBy(pod, gs) {
		return false, files.ErrForeignPod
	}
	if pod.DeletionTimestamp != nil {
		return false, nil
	}
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		return false, r.Delete(ctx, pod)
	}
	return gameserver.PodReady(pod), nil
}

// containerState returns the state of the main container.
func containerState(pod *corev1.Pod) (corev1.ContainerState, bool) {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == gameserver.ContainerName {
			return cs.State, true
		}
	}
	return corev1.ContainerState{}, false
}

// terminated reports whether the main container has exited and its exit code.
func terminated(pod *corev1.Pod) (bool, int32, string) {
	if st, ok := containerState(pod); ok && st.Terminated != nil {
		return true, st.Terminated.ExitCode, st.Terminated.Reason
	}
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		return true, -1, pod.Status.Reason
	}
	return false, 0, ""
}

// waitingProblem returns a message when the container cannot be started.
func waitingProblem(pod *corev1.Pod) string {
	st, ok := containerState(pod)
	if !ok || st.Waiting == nil {
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse && c.Message != "" {
				return "Unschedulable: " + c.Message
			}
		}
		return ""
	}
	switch st.Waiting.Reason {
	case "ErrImagePull", "ImagePullBackOff", "InvalidImageName", "CreateContainerConfigError", "CreateContainerError":
		return st.Waiting.Reason + ": " + st.Waiting.Message
	}
	return ""
}
