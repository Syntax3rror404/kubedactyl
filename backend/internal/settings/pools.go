package settings

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Pool describes a Cilium load balancer IP pool (CiliumLoadBalancerIPPool).
type Pool struct {
	Name     string `json:"name"     example:"general-pool"`
	Disabled bool   `json:"disabled"`
	// Blocks are the address ranges ("10.0.0.0/24" or "10.0.0.10-10.0.0.20").
	Blocks []string `json:"blocks" example:"192.168.1.60-192.168.1.120"`
	// ServiceLabels are set on a server's service so that the pool selects it.
	ServiceLabels map[string]string `json:"serviceLabels"`
	// Selectable is false when the pool cannot be targeted by service labels.
	Selectable bool   `json:"selectable"`
	Reason     string `json:"reason,omitempty"`
	// IPs as reported by Cilium in the pool status (-1 when unknown). Numbers, not integers:
	// an IPv6 block holds more addresses than an int64 (a /64 has 2^64).
	IPsTotal     float64 `json:"ipsTotal"`
	IPsAvailable float64 `json:"ipsAvailable"`
	IPsUsed      float64 `json:"ipsUsed"`
	Conflict     bool    `json:"conflict"`
	// Families split the addresses by IP family (IPv4 first), counted by the panel: Cilium counts both
	// together. Only the pools API and the health check fill them in.
	Families []PoolFamily `json:"families,omitempty"`

	ranges           []addrRange
	reserveFirstLast bool
}

// ErrNoCilium is returned when the cluster has no CiliumLoadBalancerIPPool resource.
var ErrNoCilium = errors.New("the cluster has no Cilium LB IPAM (CiliumLoadBalancerIPPool)")

// Served versions of the pool resource, newest first (v2alpha1 before Cilium 1.18).
var poolVersions = []string{"v2", "v2alpha1"}

// ListPools returns all load balancer IP pools, sorted by name.
func ListPools(ctx context.Context, r client.Reader) ([]Pool, error) {
	for _, v := range poolVersions {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(
			schema.GroupVersionKind{Group: "cilium.io", Version: v, Kind: "CiliumLoadBalancerIPPoolList"},
		)
		if err := r.List(ctx, list); err != nil {
			if meta.IsNoMatchError(err) {
				continue
			}
			return nil, err
		}
		out := make([]Pool, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, parsePool(&list.Items[i]))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out, nil
	}
	return nil, ErrNoCilium
}

// GetPool returns one pool by name.
func GetPool(ctx context.Context, r client.Reader, name string) (Pool, error) {
	pools, err := ListPools(ctx, r)
	if err != nil {
		return Pool{}, err
	}
	for _, p := range pools {
		if p.Name == name {
			return p, nil
		}
	}
	return Pool{}, fmt.Errorf("load balancer pool %q does not exist", name)
}

func parsePool(u *unstructured.Unstructured) Pool {
	p := Pool{
		Name:          u.GetName(),
		ServiceLabels: map[string]string{},
		IPsTotal:      -1,
		IPsAvailable:  -1,
		IPsUsed:       -1,
	}
	p.Disabled, _, _ = unstructured.NestedBool(u.Object, "spec", "disabled")
	allowFirstLast, _, _ := unstructured.NestedString(u.Object, "spec", "allowFirstLastIPs")
	p.reserveFirstLast = allowFirstLast == "No"
	p.Blocks, p.ranges = parseBlocks(u)

	p.Selectable = true
	sel, hasSel, _ := unstructured.NestedMap(u.Object, "spec", "serviceSelector")
	if exprs, _, _ := unstructured.NestedSlice(sel, "matchExpressions"); len(exprs) > 0 {
		p.Selectable, p.Reason = false, "the service selector uses match expressions"
	}
	labels, _, _ := unstructured.NestedStringMap(sel, "matchLabels")
	for k, v := range labels {
		// Cilium matches these virtual labels against the service namespace and name.
		if strings.HasPrefix(k, "io.kubernetes.service.") {
			p.Selectable, p.Reason = false, "the service selector depends on the service namespace or name"
		}
		p.ServiceLabels[k] = v
	}
	if !hasSel && p.Reason == "" {
		p.Reason = "no service selector: the pool serves every load balancer service"
	}
	if p.Disabled {
		p.Selectable, p.Reason = false, "the pool is disabled"
	}

	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		m, _ := c.(map[string]any)
		msg, _ := m["message"].(string)
		switch m["type"] {
		case "cilium.io/IPsTotal":
			p.IPsTotal = count(msg)
		case "cilium.io/IPsAvailable":
			p.IPsAvailable = count(msg)
		case "cilium.io/IPsUsed":
			p.IPsUsed = count(msg)
		case "cilium.io/PoolConflict":
			p.Conflict = m["status"] == "True"
		}
	}
	return p
}

// parseBlocks returns the blocks as shown ("10.0.0.0/24", "10.0.0.10-10.0.0.20") and their ranges.
func parseBlocks(u *unstructured.Unstructured) ([]string, []addrRange) {
	shown := []string{}
	var ranges []addrRange
	blocks, _, _ := unstructured.NestedSlice(u.Object, "spec", "blocks")
	for _, b := range blocks {
		m, _ := b.(map[string]any)
		cidr, _ := m["cidr"].(string)
		start, _ := m["start"].(string)
		stop, _ := m["stop"].(string)
		switch {
		case cidr == "" && start == "":
			continue
		case cidr != "":
			shown = append(shown, cidr)
		case stop == "" || stop == start:
			shown = append(shown, start)
		default:
			shown = append(shown, start+"-"+stop)
		}
		if r, ok := parseRange(cidr, start, stop); ok {
			ranges = append(ranges, r)
		}
	}
	return shown, ranges
}

// count reads an address count of the pool status (-1 when it is not a number). Cilium writes
// arbitrarily large integers (big.Int), so it is parsed as a big.Int and rounded to a float64.
func count(msg string) float64 {
	n, ok := new(big.Int).SetString(msg, 10)
	if !ok {
		return -1
	}
	return toFloat(n)
}

// Contains reports whether the IP is inside one of the pool's blocks.
func (p Pool) Contains(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	return err == nil && slices.ContainsFunc(p.ranges, func(r addrRange) bool { return r.contains(addr) })
}

// PoolResolver resolves pool names to service labels for the controller. Results are
// cached for a minute so reconciles do not list the pools every time.
type PoolResolver struct {
	Reader client.Reader

	mu      sync.Mutex
	pools   map[string]Pool
	fetched time.Time
}

const poolCacheTTL = time.Minute

// ServiceLabels returns the labels a service needs to get an address from the pool.
func (r *PoolResolver) ServiceLabels(ctx context.Context, name string) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.pools[name]
	if !ok || time.Since(r.fetched) > poolCacheTTL {
		list, err := ListPools(ctx, r.Reader)
		if err != nil {
			return nil, err
		}
		r.pools = map[string]Pool{}
		for _, pool := range list {
			r.pools[pool.Name] = pool
		}
		r.fetched = time.Now()
		if p, ok = r.pools[name]; !ok {
			return nil, fmt.Errorf("load balancer pool %q does not exist", name)
		}
	}
	return p.ServiceLabels, nil
}
