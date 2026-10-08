package serverctl

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/tenancy"
	"app/internal/testutil"
)

// transferFixture is a stopped, installed server of alice with a bound volume and a service.
// Deleting the server also deletes its claim and service, like the garbage collector does.
func transferFixture(t *testing.T) (*Ops, client.Client, *v1alpha1.GameServer) {
	b := testutil.Builder(t)
	ns := tenancy.Namespace("alice")
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "srv",
			Namespace: ns,
			Labels:    map[string]string{tenancy.LabelOwner: "alice"},
		},
		Spec: v1alpha1.GameServerSpec{EggRef: "paper", State: v1alpha1.PowerStopped, InstallRevision: 2},
		Status: v1alpha1.GameServerStatus{
			Phase:             v1alpha1.PhaseOffline,
			InstalledRevision: 2,
			Address:           "192.168.1.20",
			Addresses:         []string{"192.168.1.20", "2001:db8::20"},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: gameserver.PVCName("srv"), Namespace: ns},
		Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "pv-1"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-1"},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			Capacity:                      corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
			ClaimRef:                      &corev1.ObjectReference{Namespace: ns, Name: pvc.Name, UID: "old-uid"},
		},
	}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "srv", Namespace: ns}}
	c := b.WithObjects(gs, pvc, pv, svc).WithStatusSubresource(&v1alpha1.GameServer{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(
				ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption,
			) error {
				if _, ok := obj.(*v1alpha1.GameServer); ok {
					_ = c.Delete(
						ctx,
						&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: pvc.Name, Namespace: ns}},
					)
					_ = c.Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "srv", Namespace: ns}})
				}
				return c.Delete(ctx, obj, opts...)
			},
		}).Build()
	return &Ops{Client: c, Reader: c}, c, gs
}

func TestTransferMovesServerAndVolume(t *testing.T) {
	ops, c, gs := transferFixture(t)
	ctx := context.Background()
	to := tenancy.Namespace("bob")
	moved, err := ops.Transfer(ctx, gs, "bob", to)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(gs), &v1alpha1.GameServer{}); err == nil {
		t.Error("the old server object still exists")
	}
	got := &v1alpha1.GameServer{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: to, Name: "srv"}, got); err != nil {
		t.Fatal(err)
	}
	if got.Labels[tenancy.LabelOwner] != "bob" || got.Spec.EggRef != "paper" || got.Spec.InstallRevision != 2 {
		t.Errorf("moved server: %+v %+v", got.Labels, got.Spec)
	}
	if got.Spec.LoadBalancerIP != "192.168.1.20,2001:db8::20" {
		t.Errorf("both addresses must stay: %q", got.Spec.LoadBalancerIP)
	}
	if got.Annotations[gameserver.AnnotationMovedVolume] != "pv-1" ||
		got.Annotations[gameserver.AnnotationInstalledRevision] != "2" {
		t.Errorf("annotations: %v", got.Annotations)
	}
	if moved.Namespace != to {
		t.Errorf("returned server is in %q", moved.Namespace)
	}
	if claim := gameserver.PVC(got, gameserver.Options{}); claim.Spec.VolumeName != "pv-1" {
		t.Errorf("the new claim must bind the moved volume: %q", claim.Spec.VolumeName)
	}
	pv := &corev1.PersistentVolume{}
	_ = c.Get(ctx, client.ObjectKey{Name: "pv-1"}, pv)
	if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		t.Errorf("the volume must be retained: %s", pv.Spec.PersistentVolumeReclaimPolicy)
	}
	if ref := pv.Spec.ClaimRef; ref == nil || ref.Namespace != to || ref.Name != gameserver.PVCName("srv") ||
		ref.UID != "" {
		t.Errorf("the volume must be reserved for the new claim: %+v", ref)
	}
}

func TestTransferRefusals(t *testing.T) {
	ops, _, gs := transferFixture(t)
	ctx := context.Background()
	if _, err := ops.Transfer(ctx, gs, "alice", gs.Namespace); !errors.Is(err, ErrSameOwner) {
		t.Errorf("same owner: %v", err)
	}
	running := gs.DeepCopy()
	running.Spec.State = v1alpha1.PowerRunning
	if _, err := ops.Transfer(ctx, running, "bob", tenancy.Namespace("bob")); !errors.Is(err, ErrRunning) {
		t.Errorf("running: %v", err)
	}
	unbound := gs.DeepCopy()
	unbound.Name = "other"
	if _, err := ops.Transfer(ctx, unbound, "bob", tenancy.Namespace("bob")); !errors.Is(err, ErrNoVolume) {
		t.Errorf("without volume: %v", err)
	}
}
