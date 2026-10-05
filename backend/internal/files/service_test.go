package files

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/testutil"
)

// TestForeignFilesPod: a pod with the files pod's name that the server does not control is never
// used for file operations.
func TestForeignFilesPod(t *testing.T) {
	ref := Ref{Namespace: "ns", Name: "srv"}
	ready := corev1.PodStatus{
		Phase:      corev1.PodRunning,
		Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
	}
	owner := metav1.OwnerReference{
		APIVersion: v1alpha1.GroupVersion.String(), Kind: "GameServer", Name: "srv", UID: "1", Controller: ptr.To(true),
	}
	for name, refs := range map[string][]metav1.OwnerReference{
		"no owner":       nil,
		"other server":   {func() metav1.OwnerReference { o := owner; o.Name = "other"; return o }()},
		"not controller": {func() metav1.OwnerReference { o := owner; o.Controller = nil; return o }()},
		"ours":           {owner},
	} {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ref.Namespace, Name: gameserver.FilesPodName(ref.Name), OwnerReferences: refs,
			},
			Status: ready,
		}
		s := &Service{
			Activity: NewActivity(), Reader: testutil.Builder(t).WithObjects(pod).Build(),
			Trigger: func(string, string) {},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := s.EnsurePod(ctx, ref)
		cancel()
		if want := name != "ours"; errors.Is(err, ErrForeignPod) != want || (!want && err != nil) {
			t.Errorf("%s: EnsurePod = %v, foreign expected %v", name, err, want)
		}
	}
}
