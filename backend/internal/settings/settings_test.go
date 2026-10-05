package settings

import (
	"slices"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
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
- metadata: {name: cidr-pool}
  spec:
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
