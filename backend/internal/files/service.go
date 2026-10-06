package files

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
)

// Service starts the files pod of a server on demand and runs long file jobs (backups,
// restores, downloads) in the background while keeping the pod alive.
type Service struct {
	Manager  *Manager
	Activity *Activity
	// Reader reads the files pod uncached (the informer can lag by minutes).
	Reader client.Reader
	// Trigger asks the controller to reconcile a server (it creates the files pod).
	Trigger func(namespace, server string)
	// Changed is called after a background job changed the files of a server (nil: nobody is told).
	Changed func(Ref)

	jobs jobStore
}

// PodState describes the files pod of a server.
type PodState struct {
	Ready bool
	// State is Stopped, Starting, Ready or Stopping.
	State   string
	Message string
	// StopsIn is how long a ready pod stays without further file operations.
	StopsIn time.Duration

	foreign bool
}

// ErrForeignPod is returned when a pod with the name of the files pod exists that the panel did
// not create for the server: file operations would run in it.
var ErrForeignPod = errors.New("a pod with the name of the file container exists that the panel did not create")

// ours reports whether the server controls the pod (the controller sets the reference).
func ours(pod *corev1.Pod, ref Ref) bool {
	owner := metav1.GetControllerOf(pod)
	return owner != nil && owner.APIVersion == v1alpha1.GroupVersion.String() &&
		owner.Kind == "GameServer" && owner.Name == ref.Name
}

// PodState reads the files pod directly from the API server.
func (s *Service) PodState(ctx context.Context, ref Ref) PodState {
	pod := &corev1.Pod{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: gameserver.FilesPodName(ref.Name)}, pod)
	switch {
	case apierrors.IsNotFound(err):
		return PodState{State: "Stopped"}
	case err != nil:
		return PodState{State: "Stopped", Message: err.Error()}
	case pod.DeletionTimestamp != nil:
		return PodState{State: "Stopping"}
	case !ours(pod, ref):
		return PodState{State: "Stopped", Message: ErrForeignPod.Error(), foreign: true}
	}
	if gameserver.PodReady(pod) {
		stopsIn := time.Until(StopsAt(s.Activity.Last(ref), ReadySince(pod)))
		return PodState{Ready: true, State: "Ready", StopsIn: max(stopsIn, 0)}
	}
	st := PodState{State: "Starting", Message: string(pod.Status.Phase)}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			st.Message = cs.State.Waiting.Reason
		}
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status != corev1.ConditionTrue && cond.Message != "" {
			st.Message = cond.Message
		}
	}
	return st
}

// ErrStarting is returned when the files pod did not become ready in time.
var ErrStarting = errors.New("the file container is still starting, try again in a few seconds")

// EnsurePod marks the files as used, asks the controller for the pod and waits until it is
// ready (at most the context deadline).
func (s *Service) EnsurePod(ctx context.Context, ref Ref) error {
	s.Activity.Touch(ref)
	switch st := s.PodState(ctx, ref); {
	case st.foreign:
		return ErrForeignPod
	case st.Ready:
		return nil
	}
	s.Trigger(ref.Namespace, ref.Name)
	for {
		select {
		case <-ctx.Done():
			return ErrStarting
		case <-time.After(500 * time.Millisecond):
		}
		// Waiting counts as activity, the operation after it too.
		s.Activity.Touch(ref)
		if s.PodState(ctx, ref).Ready {
			return nil
		}
	}
}
