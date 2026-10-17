package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/testutil"
)

// A new migration must never take over the copy of an earlier one that was cancelled or failed.
func TestRemoveOldCopy(t *testing.T) {
	started := time.Now().Truncate(time.Second)
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "mc", Namespace: "kubedactyl-user-alice"}}
	m := &v1alpha1.StorageMigration{StartedAt: metav1.NewTime(started.Add(500 * time.Millisecond))}
	meta := func(age time.Duration) metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Name: gameserver.MigrateName(gs.Name), Namespace: gs.Namespace,
			CreationTimestamp: metav1.NewTime(started.Add(-age)),
		}
	}
	pod := func(age time.Duration) client.Object { return &corev1.Pod{ObjectMeta: meta(age)} }
	pvc := func(age time.Duration) client.Object { return &corev1.PersistentVolumeClaim{ObjectMeta: meta(age)} }
	for _, tc := range []struct {
		name      string
		objects   []client.Object
		wantClean bool
		removed   client.Object
	}{
		{"nothing left", nil, true, nil},
		{"own claim of this migration", []client.Object{pvc(0)}, true, nil},
		{"own pod and claim", []client.Object{pod(0), pvc(0)}, true, nil},
		{"pod of a cancelled migration", []client.Object{pod(time.Minute), pvc(time.Minute)}, false, &corev1.Pod{}},
		{"claim of a failed migration", []client.Object{pvc(time.Minute)}, false, &corev1.PersistentVolumeClaim{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testutil.Builder(t).WithObjects(tc.objects...).Build()
			r := &Reconciler{Client: c, Reader: c}
			clean, err := r.removeOldCopy(context.Background(), gs, m)
			if err != nil || clean != tc.wantClean {
				t.Fatalf("clean = %v, %v; want %v", clean, err, tc.wantClean)
			}
			if tc.removed == nil {
				return
			}
			key := client.ObjectKey{Namespace: gs.Namespace, Name: gameserver.MigrateName(gs.Name)}
			if err := c.Get(context.Background(), key, tc.removed); !apierrors.IsNotFound(err) {
				t.Errorf("old %T is still there", tc.removed)
			}
		})
	}
}
