package clusterinfo

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"app/internal/kube"
	"app/internal/testutil"
)

func TestLoadCache(t *testing.T) {
	const ns = "kubedactyl"
	stored := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: hardwareCacheCM, Namespace: ns},
		Data:       map[string]string{"w1": `{"hardware":{"cpuModel":"Xeon"},"probedAt":"2026-10-01T00:00:00Z"}`},
	}
	probe := func(name string, labels map[string]string) *corev1.Pod {
		return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels}}
	}
	leftover := probe("kubedactyl-probe-abcde", map[string]string{probeLabel: probeRole})
	other := probe("other", nil)
	cases := []struct {
		name       string
		objects    []runtime.Object
		readErr    error
		wantLoaded bool
		wantNodes  int
		wantPods   int // left in the namespace
	}{
		{"stored results", []runtime.Object{stored, leftover, other}, nil, true, 1, 1},
		{"nothing stored yet", []runtime.Object{leftover}, nil, true, 0, 0},
		{"read failed", []runtime.Object{stored, leftover}, errors.New("timeout"), false, 0, 1},
	}
	for _, c := range cases {
		cs := k8sfake.NewClientset(c.objects...)
		if c.readErr != nil {
			cs.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, c.readErr
			})
		}
		s := &Service{Kube: &kube.Client{Clientset: cs}, Namespace: ns, Log: testutil.Logger()}
		if loaded := s.loadCache(t.Context()); loaded != c.wantLoaded || s.loaded != c.wantLoaded {
			t.Errorf("%s: loaded %v, want %v", c.name, loaded, c.wantLoaded)
		}
		pods, _ := cs.CoreV1().Pods(ns).List(t.Context(), metav1.ListOptions{})
		if len(s.hardware) != c.wantNodes || len(pods.Items) != c.wantPods {
			t.Errorf("%s: %d results, %d pods left, want %d and %d",
				c.name, len(s.hardware), len(pods.Items), c.wantNodes, c.wantPods)
		}
	}
}

func TestForgetGoneNodes(t *testing.T) {
	s := &Service{hardware: map[string]*hardwareEntry{"w1": {}, "gone": {}}}
	s.forgetGoneNodes([]corev1.Node{node("w1", true), node("w2", true)})
	if _, ok := s.hardware["gone"]; ok || len(s.hardware) != 1 {
		t.Errorf("results = %v, want only w1", s.hardware)
	}
}

func TestProbePodNameFitsLongNodeNames(t *testing.T) {
	name := "a-very-long-node-name-that-is-close-to-the-limit-of-63-characters"
	pod := (&Service{HelperImage: "alpine"}).probePod(name)
	if pod.Name != "" || pod.GenerateName != "kubedactyl-probe-" || pod.Labels[probeLabel] != probeRole {
		t.Errorf("name %q, generate name %q, labels %v", pod.Name, pod.GenerateName, pod.Labels)
	}
}
