package settings

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strconv"
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
	// IPs as reported by Cilium in the pool status (-1 when unknown).
	IPsTotal     int64 `json:"ipsTotal"`
	IPsAvailable int64 `json:"ipsAvailable"`
	IPsUsed      int64 `json:"ipsUsed"`
	Conflict     bool  `json:"conflict"`
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
		Blocks:        []string{},
		ServiceLabels: map[string]string{},
		IPsTotal:      -1,
		IPsAvailable:  -1,
		IPsUsed:       -1,
	}
	p.Disabled, _, _ = unstructured.NestedBool(u.Object, "spec", "disabled")
	blocks, _, _ := unstructured.NestedSlice(u.Object, "spec", "blocks")
	for _, b := range blocks {
		m, _ := b.(map[string]any)
		if cidr, _ := m["cidr"].(string); cidr != "" {
			p.Blocks = append(p.Blocks, cidr)
		} else if start, _ := m["start"].(string); start != "" {
			stop, _ := m["stop"].(string)
			if stop == "" || stop == start {
				p.Blocks = append(p.Blocks, start)
			} else {
				p.Blocks = append(p.Blocks, start+"-"+stop)
			}
		}
	}

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
		n, err := strconv.ParseInt(msg, 10, 64)
		switch m["type"] {
		case "cilium.io/IPsTotal":
			p.IPsTotal = valueOr(n, err)
		case "cilium.io/IPsAvailable":
			p.IPsAvailable = valueOr(n, err)
		case "cilium.io/IPsUsed":
			p.IPsUsed = valueOr(n, err)
		case "cilium.io/PoolConflict":
			p.Conflict = m["status"] == "True"
		}
	}
	return p
}

func valueOr(n int64, err error) int64 {
	if err != nil {
		return -1
	}
	return n
}

// Contains reports whether the IP is inside one of the pool's blocks.
func (p Pool) Contains(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, b := range p.Blocks {
		if prefix, err := netip.ParsePrefix(b); err == nil {
			if prefix.Contains(addr) {
				return true
			}
			continue
		}
		start, stop, found := strings.Cut(b, "-")
		if !found {
			stop = start
		}
		lo, err1 := netip.ParseAddr(start)
		hi, err2 := netip.ParseAddr(stop)
		if err1 == nil && err2 == nil && lo.Compare(addr) <= 0 && addr.Compare(hi) <= 0 {
			return true
		}
	}
	return false
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
