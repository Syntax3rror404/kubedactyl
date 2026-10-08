package settings

import (
	"context"
	"math/big"
	"net/netip"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PoolFamily counts the addresses of one IP family of a pool. Numbers, not integers: an IPv6
// block holds more addresses than an int64.
type PoolFamily struct {
	Family    string  `json:"family"    enums:"IPv4,IPv6"`
	Total     float64 `json:"total"`
	Available float64 `json:"available"`
	Used      float64 `json:"used"`
}

// addrRange is one block of a pool, from and to included.
type addrRange struct {
	from, to netip.Addr
	cidr     bool
}

// parseRange reads a block the way Cilium does: a CIDR, or start (to stop, if given).
func parseRange(cidr, start, stop string) (addrRange, bool) {
	if cidr != "" {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return addrRange{}, false
		}
		prefix = prefix.Masked()
		return addrRange{from: prefix.Addr(), to: lastAddr(prefix), cidr: true}, true
	}
	from, err := netip.ParseAddr(start)
	if err != nil {
		return addrRange{}, false
	}
	to := from
	if stop != "" {
		if to, err = netip.ParseAddr(stop); err != nil || to.Is4() != from.Is4() || to.Less(from) {
			return addrRange{}, false
		}
	}
	return addrRange{from: from, to: to}, true
}

// lastAddr is the highest address of the prefix.
func lastAddr(prefix netip.Prefix) netip.Addr {
	b := prefix.Addr().As16()
	first := prefix.Bits()
	if prefix.Addr().Is4() {
		first += 96 // As16 maps IPv4 into the last 32 bits
	}
	for i := first; i < 128; i++ {
		b[i/8] |= 1 << (7 - i%8)
	}
	addr := netip.AddrFrom16(b)
	if prefix.Addr().Is4() {
		return addr.Unmap()
	}
	return addr
}

func (r addrRange) contains(a netip.Addr) bool {
	return r.from.Compare(a) <= 0 && a.Compare(r.to) <= 0
}

func (r addrRange) size() *big.Int {
	from, to := r.from.As16(), r.to.As16()
	n := new(big.Int).Sub(new(big.Int).SetBytes(to[:]), new(big.Int).SetBytes(from[:]))
	return n.Add(n, big.NewInt(1))
}

// ListPoolsWithUsage lists the pools with their Families filled in (one more list call, for all
// services of the cluster).
func ListPoolsWithUsage(ctx context.Context, r client.Reader) ([]Pool, error) {
	pools, err := ListPools(ctx, r)
	if err != nil {
		return nil, err
	}
	return pools, countAddresses(ctx, r, pools)
}

// countAddresses fills in the Families of the pools. Cilium reports one count for both families,
// so the panel counts itself: the addresses of the blocks, and as used the load balancer IPs of
// all services in the cluster plus the first and last IP of CIDR blocks the pool keeps back.
func countAddresses(ctx context.Context, r client.Reader, pools []Pool) error {
	var list corev1.ServiceList
	if err := r.List(ctx, &list); err != nil {
		return err
	}
	used := map[netip.Addr]bool{}
	for _, svc := range list.Items {
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if addr, err := netip.ParseAddr(ing.IP); err == nil {
				used[addr] = true
			}
		}
	}
	for i := range pools {
		pools[i].Families = pools[i].families(used)
	}
	return nil
}

func (p Pool) families(used map[netip.Addr]bool) []PoolFamily {
	var out []PoolFamily
	for _, family := range []string{"IPv4", "IPv6"} {
		total, taken := new(big.Int), new(big.Int)
		found := false
		for _, r := range p.ranges {
			if r.from.Is4() != (family == "IPv4") {
				continue
			}
			found = true
			n := r.size()
			total.Add(total, n)
			// Cilium keeps back the first and last IP only for CIDRs with more than two addresses.
			if p.reserveFirstLast && r.cidr && n.Cmp(big.NewInt(2)) > 0 {
				taken.Add(taken, big.NewInt(2))
			}
			for addr := range used {
				if r.contains(addr) {
					taken.Add(taken, big.NewInt(1))
				}
			}
		}
		if found {
			out = append(out, PoolFamily{
				Family: family, Total: toFloat(total), Available: toFloat(new(big.Int).Sub(total, taken)),
				Used: toFloat(taken),
			})
		}
	}
	return out
}

func toFloat(n *big.Int) float64 {
	f, _ := new(big.Float).SetInt(n).Float64()
	return f
}

// CiliumConfig holds the settings of the Cilium agents (where Cilium's Helm chart puts them).
var CiliumConfig = client.ObjectKey{Namespace: "kube-system", Name: "cilium-config"}

// IPv6Missing says why services of the cluster get no IPv6 address, or "" when they can or it is
// unknown (Kubernetes before 1.33 has no ServiceCIDR objects, Cilium may keep its config elsewhere).
// The pool having an IPv6 block is not enough: the API server keeps only the IP families of its
// service CIDRs on a service, and Cilium asks for IPv6 only with IPv6 switched on.
func IPv6Missing(ctx context.Context, r client.Reader) string {
	var reasons []string
	var cidrs networkingv1.ServiceCIDRList
	if err := r.List(ctx, &cidrs); err == nil && len(cidrs.Items) > 0 && !anyIPv6(cidrs.Items) {
		reasons = append(reasons, "No IPv6 service CIDR")
	}
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, CiliumConfig, cm); err == nil && cm.Data["enable-ipv6"] == "false" {
		reasons = append(reasons, "IPv6 off in Cilium")
	}
	return strings.Join(reasons, " · ")
}

func anyIPv6(cidrs []networkingv1.ServiceCIDR) bool {
	for _, c := range cidrs {
		for _, cidr := range c.Spec.CIDRs {
			if prefix, err := netip.ParsePrefix(cidr); err == nil && prefix.Addr().Is6() {
				return true
			}
		}
	}
	return false
}
