package settings

import (
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/kube"
	"app/internal/testutil"
)

func TestKubeAPIQPS(t *testing.T) {
	ctx := t.Context()
	longhorn := &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "longhorn"}}
	c := testutil.Builder(t).WithObjects(longhorn).Build()
	limiter := kube.NewRateLimiter(DefaultKubeAPIQPS, DefaultKubeAPIUserQPS)
	spec := func(qps int32) v1alpha1.PanelSettingsSpec {
		return v1alpha1.PanelSettingsSpec{StorageClasses: []string{"longhorn"}, KubeAPIQPS: qps}
	}
	s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace, Limiter: limiter}

	if spec, err := s.Get(ctx); err != nil || spec.KubeAPIQPS != DefaultKubeAPIQPS {
		t.Fatalf("default: %d %v", spec.KubeAPIQPS, err)
	}
	for _, bad := range []int32{4, 1001} {
		if _, err := s.Update(ctx, spec(bad)); err == nil {
			t.Errorf("%d requests per second accepted", bad)
		}
	}
	if _, err := s.Update(ctx, spec(120)); err != nil {
		t.Fatal(err)
	}
	if limiter.QPS() != 120 {
		t.Errorf("saved limit not applied: %v", limiter.QPS())
	}
	if spec, _ := s.Get(ctx); spec.KubeAPIQPS != 120 {
		t.Errorf("stored limit: %d", spec.KubeAPIQPS)
	}
}
