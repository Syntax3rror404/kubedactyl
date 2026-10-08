package settings

import (
	"math"
	"net/netip"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/internal/testutil"
)

func loadBalancer(name string, ips ...string) *corev1.Service {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "games"}}
	for _, ip := range ips {
		svc.Status.LoadBalancer.Ingress = append(svc.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{IP: ip})
	}
	return svc
}

func TestCountAddresses(t *testing.T) {
	c := testutil.Builder(t).WithObjects(
		loadBalancer("dual", "192.0.2.161", "2001:db8::1"),
		loadBalancer("shared", "192.0.2.161"), // a shared IP counts once
		loadBalancer("cidr", "10.10.0.5"),
		loadBalancer("elsewhere", "198.51.100.1"),
		loadBalancer("pending"),
	).Build()
	byName := testPools(t)
	pools := []Pool{byName["dual-stack-pool"], byName["cidr-pool"], byName["general-pool"]}
	if err := countAddresses(t.Context(), c, pools); err != nil {
		t.Fatal(err)
	}

	dual := pools[0].Families
	if len(dual) != 2 || dual[0] != (PoolFamily{Family: "IPv4", Total: 32, Available: 31, Used: 1}) {
		t.Fatalf("dual-stack-pool = %+v", dual)
	}
	v6 := dual[1]
	if v6.Family != "IPv6" || v6.Total != math.Pow(2, 64) || v6.Used != 1 || v6.Available < 1.8e19 {
		t.Errorf("dual-stack-pool IPv6 = %+v", v6)
	}
	// 16 + 1 addresses; the /28 keeps its first and last IP back (allowFirstLastIPs: No).
	if cidr := pools[1].Families; len(cidr) != 1 ||
		cidr[0] != (PoolFamily{Family: "IPv4", Total: 17, Available: 14, Used: 3}) {
		t.Errorf("cidr-pool = %+v", cidr)
	}
	if general := pools[2].Families; len(general) != 1 || general[0].Total != 61 || general[0].Used != 0 {
		t.Errorf("general-pool = %+v", general)
	}
}

func TestParseRange(t *testing.T) {
	for _, tc := range []struct{ cidr, start, stop, from, to string }{
		{cidr: "192.0.2.160/27", from: "192.0.2.160", to: "192.0.2.191"},
		{cidr: "192.0.2.170/27", from: "192.0.2.160", to: "192.0.2.191"},
		{cidr: "2001:db8:0:1337::/64", from: "2001:db8:0:1337::", to: "2001:db8:0:1337:ffff:ffff:ffff:ffff"},
		{start: "10.0.0.1", from: "10.0.0.1", to: "10.0.0.1"},
		{start: "2001:db8::1", stop: "2001:db8::ff", from: "2001:db8::1", to: "2001:db8::ff"},
	} {
		r, ok := parseRange(tc.cidr, tc.start, tc.stop)
		if !ok || r.from != netip.MustParseAddr(tc.from) || r.to != netip.MustParseAddr(tc.to) {
			t.Errorf("%+v: %v %v-%v", tc, ok, r.from, r.to)
		}
	}
	bad := [][3]string{{"nonsense", "", ""}, {"", "10.0.0.9", "10.0.0.1"}, {"", "10.0.0.1", "2001:db8::1"}}
	for _, bad := range bad {
		if _, ok := parseRange(bad[0], bad[1], bad[2]); ok {
			t.Errorf("%v must not parse", bad)
		}
	}
}

func TestIPv6Missing(t *testing.T) {
	cidr := func(cidrs ...string) *networkingv1.ServiceCIDR {
		return &networkingv1.ServiceCIDR{
			ObjectMeta: metav1.ObjectMeta{Name: "kubernetes"}, Spec: networkingv1.ServiceCIDRSpec{CIDRs: cidrs},
		}
	}
	cilium := func(ipv6 string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: CiliumConfig.Name, Namespace: CiliumConfig.Namespace},
			Data:       map[string]string{"enable-ipv6": ipv6},
		}
	}
	for name, tc := range map[string]struct {
		objects []client.Object
		want    string
	}{
		"IPv4 only": {
			[]client.Object{cidr("10.245.0.0/16"), cilium("false")}, "No IPv6 service CIDR · IPv6 off in Cilium",
		},
		"dual stack": {[]client.Object{cidr("10.245.0.0/16", "fd00:10:245::/108"), cilium("true")}, ""},
		"Cilium without IPv6": {
			[]client.Object{cidr("10.245.0.0/16", "fd00:10:245::/108"), cilium("false")}, "IPv6 off in Cilium",
		},
		"unknown": {nil, ""},
	} {
		c := testutil.Builder(t).WithObjects(tc.objects...).Build()
		if got := IPv6Missing(t.Context(), c); got != tc.want {
			t.Errorf("%s: %q, want %q", name, got, tc.want)
		}
	}
}
