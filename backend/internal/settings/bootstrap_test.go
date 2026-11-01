package settings

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/tenancy"
	"app/internal/testutil"
)

// TestMigrateServers: servers get the storage class of their volume; servers with nothing to fill in are not
// changed.
func TestMigrateServers(t *testing.T) {
	ns := tenancy.Namespace("alice")
	server := func(name, storageClass string) *v1alpha1.GameServer {
		return &v1alpha1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec:       v1alpha1.GameServerSpec{StorageClass: storageClass},
		}
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: gameserver.PVCName("old"), Namespace: ns},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: ptr.To("longhorn")},
	}
	c := testutil.Builder(t).WithObjects(server("old", ""), server("unknown", ""), pvc).Build()
	before := map[string]string{}
	for _, name := range []string{"old", "unknown"} {
		gs := &v1alpha1.GameServer{}
		_ = c.Get(t.Context(), client.ObjectKey{Namespace: ns, Name: name}, gs)
		before[name] = gs.ResourceVersion
	}
	if err := migrateServers(t.Context(), c, v1alpha1.PanelSettingsSpec{}, testutil.Logger()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, storageClass string
		changed            bool
	}{
		{"old", "longhorn", true},
		{"unknown", "", false},
	}
	for _, tc := range cases {
		gs := &v1alpha1.GameServer{}
		_ = c.Get(t.Context(), client.ObjectKey{Namespace: ns, Name: tc.name}, gs)
		if gs.Spec.StorageClass != tc.storageClass || (gs.ResourceVersion != before[tc.name]) != tc.changed {
			t.Errorf("%s: storage class %q, changed %t", tc.name, gs.Spec.StorageClass, !tc.changed)
		}
	}
}
