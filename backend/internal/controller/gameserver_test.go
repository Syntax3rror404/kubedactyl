package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"app/api/v1alpha1"
	"app/internal/console"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
	"app/internal/testutil"
)

// testServer is a server of alice that should run; the egg "paper" has no done line.
func testServer() *v1alpha1.GameServer {
	return &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "mc", Namespace: "kubedactyl-user-alice", UID: "server-uid"},
		Spec: v1alpha1.GameServerSpec{
			EggRef: "paper", State: v1alpha1.PowerRunning, StopTimeoutSeconds: 30,
			Resources: v1alpha1.Resources{DiskMiB: 1024},
		},
	}
}

func testEgg() *v1alpha1.Egg {
	return &v1alpha1.Egg{
		ObjectMeta: metav1.ObjectMeta{Name: "paper", Namespace: testutil.Namespace},
		Spec:       v1alpha1.EggSpec{Stop: "^SIGTERM"},
	}
}

// gamePod is the game pod of testServer with the given main container state.
func gamePod(uid types.UID, state corev1.ContainerState) *corev1.Pod {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: gameserver.GamePodName("mc"), Namespace: "kubedactyl-user-alice", UID: uid,
	}}
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: gameserver.ContainerName, State: state}}
	return pod
}

// testReconciler builds a reconciler on a fake client; the console hub streams from a fake clientset.
func testReconciler(t *testing.T, b *interceptor.Funcs, objects ...client.Object) (*Reconciler, client.Client) {
	builder := testutil.Builder(t).WithObjects(objects...).WithStatusSubresource(&v1alpha1.GameServer{})
	if b != nil {
		builder = builder.WithInterceptorFuncs(*b)
	}
	c := builder.Build()
	hub := console.NewHub(&kube.Client{Clientset: k8sfake.NewClientset()}, testutil.Logger())
	return &Reconciler{
		Client: c, Reader: c, Hub: hub, Log: testutil.Logger(),
		Opts:  gameserver.Options{Namespace: testutil.Namespace},
		Files: &files.Service{Activity: files.NewActivity()},
	}, c
}

// The status is saved as a merge patch of what the reconcile changed: it never conflicts with
// writes of others and keeps their fields, and cleared fields are removed.
func TestSaveStatus(t *testing.T) {
	ctx := context.Background()
	stored := testServer()
	stored.Status = v1alpha1.GameServerStatus{
		Phase: v1alpha1.PhaseStopping, PodUID: "pod-1", StopRequestedAt: &metav1.Time{Time: time.Now()},
		Addresses: []string{"192.168.1.20", "2001:db8::20"}, Address: "192.168.1.20",
	}
	r, c := testReconciler(t, nil, stored)
	read := &v1alpha1.GameServer{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(stored), read); err != nil {
		t.Fatal(err)
	}
	gs := read.DeepCopy()
	before := gs.Status.DeepCopy()
	gs.Status.Phase, gs.Status.PodUID, gs.Status.StopRequestedAt = v1alpha1.PhaseOffline, "", nil
	gs.Status.Addresses, gs.Status.Address = nil, ""
	gs.Status.LastCrashAt = &metav1.Time{Time: time.Now()}
	// A spec patch during the reconcile keeps the built status.
	if err := r.patchServer(ctx, gs, func() { gs.Spec.State = v1alpha1.PowerStopped }); err != nil {
		t.Fatal(err)
	}
	// Written by the schedule runner meanwhile: the reconcile's resource version is old now.
	other := &v1alpha1.GameServer{}
	_ = c.Get(ctx, client.ObjectKeyFromObject(stored), other)
	other.Status.Schedules = []v1alpha1.ScheduleStatus{{Name: "backup", LastResult: "ok"}}
	if err := c.Status().Update(ctx, other); err != nil {
		t.Fatal(err)
	}
	if err := r.saveStatus(ctx, gs, before); err != nil {
		t.Fatalf("saving the status: %v", err)
	}

	got := &v1alpha1.GameServer{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(stored), got); err != nil {
		t.Fatal(err)
	}
	st := got.Status
	if st.Phase != v1alpha1.PhaseOffline || st.LastCrashAt == nil || got.Spec.State != v1alpha1.PowerStopped {
		t.Errorf("changes lost: phase %q, lastCrashAt %v, state %q", st.Phase, st.LastCrashAt, got.Spec.State)
	}
	if st.PodUID != "" || st.StopRequestedAt != nil || st.Addresses != nil || st.Address != "" {
		t.Errorf("cleared fields kept: %+v", st)
	}
	if len(st.Schedules) != 1 {
		t.Errorf("the schedule result written meanwhile is lost: %+v", st.Schedules)
	}
}

// A volume that cannot grow (storage class without volume expansion) must not stop the reconcile.
func TestEnsurePVCResize(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rejected bool
		wantSize string
	}{
		{"grows", false, "1Gi"},
		{"resize rejected", true, "512Mi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := testServer()
			pvc := gameserver.PVC(gs, gameserver.Options{StorageClass: "longhorn"})
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("512Mi")
			funcs := &interceptor.Funcs{Patch: func(
				ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch,
				opts ...client.PatchOption,
			) error {
				if _, ok := obj.(*corev1.PersistentVolumeClaim); ok && tc.rejected {
					return apierrors.NewForbidden(schema.GroupResource{Resource: "persistentvolumeclaims"},
						obj.GetName(), errors.New("the storage class does not allow volume expansion"))
				}
				return c.Patch(ctx, obj, patch, opts...)
			}}
			r, c := testReconciler(t, funcs, gs, pvc)
			if err := r.ensurePVC(context.Background(), gs); err != nil {
				t.Fatalf("ensurePVC: %v", err)
			}
			if gs.Status.StorageClass != "longhorn" {
				t.Errorf("storage class not recorded: %q", gs.Status.StorageClass)
			}
			got := &corev1.PersistentVolumeClaim{}
			_ = c.Get(context.Background(), client.ObjectKeyFromObject(pvc), got)
			size := got.Spec.Resources.Requests[corev1.ResourceStorage]
			if size.Cmp(resource.MustParse(tc.wantSize)) != 0 {
				t.Errorf("size = %s, want %s", size.String(), tc.wantSize)
			}
		})
	}
}

// A finished pod goes terminatedGrace after its container exited.
func TestRemoveTerminated(t *testing.T) {
	exited := func(ago time.Duration) *corev1.Pod {
		return gamePod("1", corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			FinishedAt: metav1.NewTime(time.Now().Add(-ago)),
		}})
	}
	deleting := exited(time.Hour)
	deleting.DeletionTimestamp, deleting.Finalizers = &metav1.Time{Time: time.Now()}, []string{"test"}
	for _, tc := range []struct {
		name     string
		pod      *corev1.Pod
		wantWait bool
		wantGone bool
	}{
		{"just exited", exited(time.Second), true, false},
		{"exit in the future (node clock ahead)", exited(-time.Hour), true, false},
		{"exited long ago", exited(time.Minute), false, true},
		{"exit time unknown", gamePod("1", corev1.ContainerState{}), false, true},
		{"being deleted", deleting, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, c := testReconciler(t, nil, tc.pod)
			wait, err := r.removeTerminated(context.Background(), tc.pod)
			if err != nil || (wait > 0) != tc.wantWait || wait > terminatedGrace {
				t.Fatalf("wait = %v, %v; want waiting %v", wait, err, tc.wantWait)
			}
			err = c.Get(context.Background(), client.ObjectKeyFromObject(tc.pod), &corev1.Pod{})
			if apierrors.IsNotFound(err) != tc.wantGone {
				t.Errorf("pod gone = %v, want %v", apierrors.IsNotFound(err), tc.wantGone)
			}
		})
	}
}
