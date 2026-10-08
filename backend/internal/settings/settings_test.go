package settings

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"app/api/v1alpha1"
	"app/internal/testutil"
)

// Shaped like the pools of the homelab cluster (Cilium 1.18, cilium.io/v2).
const poolsYAML = `
- metadata: {name: general-pool}
  spec:
    blocks: [{start: 192.168.1.60, stop: 192.168.1.120}]
    disabled: false
    serviceSelector: {matchLabels: {lb.cilium.io/pool: general}}
  status:
    conditions:
    - {type: cilium.io/PoolConflict, status: "False", message: ""}
    - {type: cilium.io/IPsTotal, status: Unknown, message: "61"}
    - {type: cilium.io/IPsAvailable, status: Unknown, message: "55"}
    - {type: cilium.io/IPsUsed, status: Unknown, message: "6"}
- metadata: {name: dual-stack-pool}
  spec:
    blocks: [{cidr: 192.0.2.160/27}, {cidr: "2001:db8::/64"}]
  status:
    conditions:
    - {type: cilium.io/IPsTotal, status: Unknown, message: "18446744073709551648"}
    - {type: cilium.io/IPsAvailable, status: Unknown, message: "18446744073709551647"}
    - {type: cilium.io/IPsUsed, status: Unknown, message: "1"}
- metadata: {name: cidr-pool}
  spec:
    allowFirstLastIPs: "No"
    blocks: [{cidr: 10.10.0.0/28}, {start: 10.20.0.1}]
- metadata: {name: expr-pool}
  spec:
    blocks: [{cidr: 10.30.0.0/28}]
    serviceSelector: {matchExpressions: [{key: tier, operator: In, values: [a]}]}
- metadata: {name: ns-pool}
  spec:
    blocks: [{cidr: 10.40.0.0/28}]
    serviceSelector: {matchLabels: {io.kubernetes.service.namespace: games}}
- metadata: {name: off-pool}
  spec:
    disabled: true
    serviceSelector: {matchLabels: {pool: off}}
`

func testPools(t *testing.T) map[string]Pool {
	t.Helper()
	var raw []map[string]any
	if err := yaml.Unmarshal([]byte(poolsYAML), &raw); err != nil {
		t.Fatal(err)
	}
	out := map[string]Pool{}
	for _, obj := range raw {
		p := parsePool(&unstructured.Unstructured{Object: obj})
		out[p.Name] = p
	}
	return out
}

func TestParsePool(t *testing.T) {
	pools := testPools(t)

	g := pools["general-pool"]
	if !g.Selectable || g.ServiceLabels["lb.cilium.io/pool"] != "general" ||
		g.Blocks[0] != "192.168.1.60-192.168.1.120" {
		t.Errorf("general-pool = %+v", g)
	}
	if g.IPsTotal != 61 || g.IPsAvailable != 55 || g.IPsUsed != 6 || g.Conflict {
		t.Errorf("general-pool usage = %+v", g)
	}
	if !g.Contains("192.168.1.60") || !g.Contains("192.168.1.120") || g.Contains("192.168.1.121") ||
		g.Contains("bogus") {
		t.Error("range containment wrong")
	}

	// Cilium counts the addresses of an IPv6 block too: more than an int64 holds.
	if d := pools["dual-stack-pool"]; d.IPsTotal < 1.8e19 || d.IPsAvailable < 1.8e19 || d.IPsUsed != 1 {
		t.Errorf("dual-stack-pool usage = %+v", d)
	}

	c := pools["cidr-pool"]
	if !c.Selectable || len(c.ServiceLabels) != 0 || c.Reason == "" || c.IPsTotal != -1 {
		t.Errorf("cidr-pool (no selector) = %+v", c)
	}
	if !c.Contains("10.10.0.15") || c.Contains("10.10.0.16") || !c.Contains("10.20.0.1") {
		t.Error("cidr/single address containment wrong")
	}

	for _, name := range []string{"expr-pool", "ns-pool", "off-pool"} {
		if p := pools[name]; p.Selectable || p.Reason == "" {
			t.Errorf("%s must not be selectable: %+v", name, p)
		}
	}
}

func TestPoolForLabels(t *testing.T) {
	var list []Pool
	for _, p := range testPools(t) {
		list = append(list, p)
	}
	if got := poolForLabels(
		list,
		map[string]string{"lb.cilium.io/pool": "general", "other": "x"},
	); got != "general-pool" {
		t.Errorf("got %q", got)
	}
	// Pools without a selector never claim a service during migration.
	if got := poolForLabels(list, map[string]string{"app": "x"}); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestChooseAndDefaults(t *testing.T) {
	allowed := []string{"a", "b"}
	if v, err := Choose("", "a", allowed, "pool"); v != "a" || err != nil {
		t.Errorf("default: %q %v", v, err)
	}
	if v, err := Choose("b", "a", allowed, "pool"); v != "b" || err != nil {
		t.Errorf("allowed: %q %v", v, err)
	}
	if _, err := Choose("c", "a", allowed, "pool"); err == nil {
		t.Error("not enabled value accepted")
	}
	if got := pickDefault("x", []string{"b", "a"}); got != "b" {
		t.Errorf("pickDefault = %q", got)
	}
	if got := unique([]string{" a", "a", "", "b"}); len(got) != 2 {
		t.Errorf("unique = %v", got)
	}
}

func TestNormalizeEggLibraries(t *testing.T) {
	got, err := normalizeEggLibraries([]string{
		" https://github.com/pterodactyl/game-eggs/ ", "https://github.com/pelican-eggs/minecraft.git", "",
		"https://github.com/pterodactyl/game-eggs",
	})
	want := []string{"https://github.com/pterodactyl/game-eggs", "https://github.com/pelican-eggs/minecraft"}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %v %v, want %v", got, err, want)
	}
	for _, bad := range []string{
		"https://gitlab.com/a/b", "http://github.com/a/b", "https://github.com/a", "https://github.com/a/b/tree/main",
	} {
		if _, err := normalizeEggLibraries([]string{bad}); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// countingReader counts the reads that would reach the API server.
type countingReader struct {
	client.Reader
	gets atomic.Int32
}

func (r *countingReader) Get(
	ctx context.Context, key client.ObjectKey, obj client.Object, o ...client.GetOption,
) error {
	r.gets.Add(1)
	return r.Reader.Get(ctx, key, obj, o...)
}

func TestCurrentKeepsSettingsUntilTheyChange(t *testing.T) {
	ctx := t.Context()
	obj := &v1alpha1.PanelSettings{
		ObjectMeta: metav1.ObjectMeta{Name: v1alpha1.SettingsName, Namespace: testutil.Namespace},
		Spec:       v1alpha1.PanelSettingsSpec{BrandName: "Acme", StorageClasses: []string{"longhorn"}},
	}
	longhorn := &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "longhorn"}}
	c := testutil.Builder(t).WithObjects(obj, longhorn).Build()
	r := &countingReader{Reader: c}
	s := &Store{Client: c, Reader: r, Namespace: testutil.Namespace}
	informers := &informertest.FakeInformers{Scheme: c.Scheme()}
	if err := s.Watch(ctx, informers); err != nil {
		t.Fatal(err)
	}
	brand := func() string {
		spec, err := s.Current(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return spec.BrandName
	}
	for range 5 {
		brand()
	}
	if n := r.gets.Load(); n != 1 {
		t.Fatalf("5 calls read the API server %d times, want 1", n)
	}

	// Changed with kubectl: the informer reports it, the next call reads once.
	changed := obj.DeepCopy()
	changed.Spec.BrandName = "Changed"
	if err := c.Update(ctx, changed); err != nil {
		t.Fatal(err)
	}
	if got := brand(); got != "Acme" {
		t.Fatalf("kept settings: %q", got)
	}
	inf, _ := informers.FakeInformerFor(ctx, &v1alpha1.PanelSettings{})
	other := changed.DeepCopy()
	other.Namespace = "another-installation"
	inf.Update(other, other)
	if got := brand(); got != "Acme" {
		t.Errorf("settings of another installation dropped the kept ones: %q", got)
	}
	inf.Update(obj, changed)
	if got := brand(); got != "Changed" || r.gets.Load() != 2 {
		t.Errorf("after the informer event: %q, %d reads", got, r.gets.Load())
	}

	// Saved through the store: applies at once.
	saved := v1alpha1.PanelSettingsSpec{BrandName: "Saved", StorageClasses: []string{"longhorn"}}
	if _, err := s.Update(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if got := brand(); got != "Saved" {
		t.Errorf("after a save: %q", got)
	}
}
