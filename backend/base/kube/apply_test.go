package kube

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateOrPatch(t *testing.T) {
	api := fake.NewClientBuilder().Build()
	cm := func() *corev1.ConfigMap {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "ns"}}
	}
	set := func(obj *corev1.ConfigMap, v string) func() error {
		return func() error { obj.Data = map[string]string{"k": v}; return nil }
	}
	obj := cm()
	if err := CreateOrPatch(t.Context(), api, api, obj, set(obj, "1")); err != nil {
		t.Fatalf("create: %v", err)
	}
	// A second call must patch, not create again (which would fail with AlreadyExists).
	obj = cm()
	if err := CreateOrPatch(t.Context(), api, api, obj, set(obj, "2")); err != nil {
		t.Fatalf("patch: %v", err)
	}
	got := &corev1.ConfigMap{}
	_ = api.Get(t.Context(), client.ObjectKey{Namespace: "ns", Name: "x"}, got)
	if got.Data["k"] != "2" {
		t.Errorf("data = %v", got.Data)
	}
	rv := got.ResourceVersion
	obj = cm()
	if err := CreateOrPatch(t.Context(), api, api, obj, set(obj, "2")); err != nil {
		t.Fatal(err)
	}
	_ = api.Get(t.Context(), client.ObjectKey{Namespace: "ns", Name: "x"}, got)
	if got.ResourceVersion != rv {
		t.Error("unchanged object must not be patched")
	}
}
