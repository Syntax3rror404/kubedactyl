package diagnostics

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"app/api/v1alpha1"
	"app/internal/checks"
)

type fakeResolver map[string][]netip.Addr // key: network ("ip4"/"ip6")

func (f fakeResolver) LookupNetIP(_ context.Context, network, _ string) ([]netip.Addr, error) {
	if ips := f[network]; len(ips) > 0 {
		return ips, nil
	}
	return nil, errors.New("no such host")
}

func ips(s ...string) []netip.Addr {
	out := make([]netip.Addr, len(s))
	for i, v := range s {
		out[i] = netip.MustParseAddr(v)
	}
	return out
}

func server(mutate func(*v1alpha1.GameServer)) *v1alpha1.GameServer {
	used := int64(100 << 20)
	gs := &v1alpha1.GameServer{
		Spec: v1alpha1.GameServerSpec{
			Ports:            []int32{25565},
			LoadBalancerPool: "general-pool",
			State:            v1alpha1.PowerRunning,
			Resources:        v1alpha1.Resources{MemoryMiB: 1024, DiskMiB: 1024},
		},
		Status: v1alpha1.GameServerStatus{Phase: v1alpha1.PhaseRunning, Address: "192.168.1.70", DiskUsedBytes: &used},
	}
	if mutate != nil {
		mutate(gs)
	}
	return gs
}

// accept opens connections to the given hosts (all when none are given) and refuses the rest.
func accept(hosts ...string) func(context.Context, string, string) (net.Conn, error) {
	return func(_ context.Context, _, address string) (net.Conn, error) {
		host, _, _ := net.SplitHostPort(address)
		if len(hosts) == 0 || slices.Contains(hosts, host) {
			c, _ := net.Pipe()
			return c, nil
		}
		return nil, errors.New("i/o timeout")
	}
}

func refused(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("dial tcp: connect: connection refused")
}

func TestRun(t *testing.T) {
	code137, code1 := int32(137), int32(1)
	full := int64(1020 << 20)
	cases := []struct {
		name     string
		gs       *v1alpha1.GameServer
		domain   string
		dns      fakeResolver
		dial     func(context.Context, string, string) (net.Conn, error)
		id       string
		status   checks.Status
		contains string
	}{
		{
			"install failed",
			server(func(g *v1alpha1.GameServer) { g.Status.Phase = v1alpha1.PhaseInstallFailed }),
			"",
			nil,
			nil,
			"install",
			checks.Error,
			"install script failed",
		},
		{
			"out of memory",
			server(
				func(g *v1alpha1.GameServer) {
					g.Status.Phase = v1alpha1.PhaseOffline
					g.Status.LastExitCode = &code137
				},
			),
			"",
			nil,
			nil,
			"state",
			checks.Error,
			"memory",
		},
		{
			"crashed",
			server(
				func(g *v1alpha1.GameServer) { g.Status.Phase = v1alpha1.PhaseOffline; g.Status.LastExitCode = &code1 },
			),
			"",
			nil,
			nil,
			"state",
			checks.Error,
			"exit code 1",
		},
		{"stopped", server(func(g *v1alpha1.GameServer) {
			g.Status.Phase = v1alpha1.PhaseOffline
			g.Spec.State = v1alpha1.PowerStopped
		}), "", nil, nil, "state", checks.Warning, "stopped"},
		{
			"suspended",
			server(func(g *v1alpha1.GameServer) { g.Spec.Suspended = true }),
			"",
			nil,
			nil,
			"state",
			checks.Error,
			"suspended",
		},
		{
			"restart required",
			server(func(g *v1alpha1.GameServer) { g.Status.RestartRequired = true }),
			"",
			nil,
			nil,
			"restart",
			checks.Warning,
			"restart",
		},
		{
			"disk full",
			server(func(g *v1alpha1.GameServer) { g.Status.DiskUsedBytes = &full }),
			"",
			nil,
			nil,
			"disk",
			checks.Error,
			"full",
		},
		{
			"no address",
			server(func(g *v1alpha1.GameServer) { g.Status.Address = "" }),
			"",
			nil,
			nil,
			"address",
			checks.Error,
			"general-pool",
		},
		{"no domain", server(nil), "", nil, nil, "domain", checks.Skipped, "No external domain"},
		{
			"cluster only",
			server(func(g *v1alpha1.GameServer) { g.Spec.ServiceType = v1alpha1.ServiceClusterIP }),
			"play.example.com",
			nil,
			nil,
			"domain",
			checks.Skipped,
			"no load balancer",
		},
		{"no A record", server(nil), "play.example.com", fakeResolver{}, nil, "dns-a", checks.Error, "no A record"},
		{
			"A to load balancer",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("192.168.1.70")},
			nil,
			"dns-a",
			checks.Error,
			"internal address",
		},
		{
			"A to wrong private IP",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("192.168.1.99")},
			nil,
			"dns-a",
			checks.Error,
			"internal address",
		},
		{
			"A public",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7")},
			refused,
			"dns-a",
			checks.OK,
			"public address",
		},
		{
			"AAAA present",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7"), "ip6": ips("2001:db8::7")},
			refused,
			"dns-aaaa",
			checks.Warning,
			"IPv6",
		},
		{
			"no AAAA",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7")},
			refused,
			"dns-aaaa",
			checks.OK,
			"IPv4",
		},
		{
			"game without TCP (UDP only)",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7")},
			refused,
			"router-25565",
			checks.Skipped,
			"only UDP",
		},
		{
			"router forwards",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7")},
			accept(),
			"router-25565",
			checks.OK,
			"answers",
		},
		{
			"TCP inside, not through the router",
			server(nil),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7")},
			accept("192.168.1.70"),
			"router-25565",
			checks.Warning,
			"not through the public address",
		},
		{
			"router skipped while stopped",
			server(func(g *v1alpha1.GameServer) { g.Status.Phase = v1alpha1.PhaseOffline }),
			"play.example.com",
			fakeResolver{"ip4": ips("203.0.113.7")},
			nil,
			"router-25565",
			checks.Skipped,
			"Start the server",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Diagnoser{Resolver: tc.dns, PublicResolver: tc.dns, Dial: tc.dial}
			list := d.Run(context.Background(), Input{Server: tc.gs, Domain: tc.domain, ShowAddresses: true})
			for _, c := range list {
				if c.ID == tc.id {
					if c.Status != tc.status || !strings.Contains(c.Message, tc.contains) {
						t.Errorf(
							"%s: got %s %q, want %s containing %q", tc.id, c.Status, c.Message, tc.status, tc.contains,
						)
					}
					return
				}
			}
			t.Errorf("no check %s in %+v", tc.id, list)
		})
	}
}

// Users never see the load balancer IP.
func TestHidesAddressForUsers(t *testing.T) {
	d := &Diagnoser{
		Resolver:       fakeResolver{"ip4": ips("192.168.1.52")},
		PublicResolver: fakeResolver{"ip4": ips("192.168.1.70")},
	}
	for _, c := range d.Run(context.Background(), Input{Server: server(nil), Domain: "play.example.com"}) {
		if strings.Contains(c.Message, "192.168.1.70") {
			t.Errorf("%s reveals the load balancer IP: %s", c.ID, c.Message)
		}
	}
}

// Local DNS servers often override the own domain; players on the internet only see the public
// records, so those decide; the local answer is not shown at all.
func TestPublicDNSDecides(t *testing.T) {
	d := &Diagnoser{
		Resolver:       fakeResolver{"ip4": ips("192.168.1.52")},
		PublicResolver: fakeResolver{"ip4": ips("203.0.113.7")},
		Dial:           accept("192.168.1.70"),
	}
	byID := map[string]checks.Check{}
	in := Input{Server: server(nil), Domain: "play.example.com", ShowAddresses: true}
	for _, c := range d.Run(context.Background(), in) {
		byID[c.ID] = c
		if strings.Contains(c.Message, "192.168.1.52") {
			t.Errorf("%s shows the local override: %s", c.ID, c.Message)
		}
	}
	if byID["dns-a"].Status != checks.OK || !strings.Contains(byID["dns-a"].Message, "203.0.113.7") {
		t.Errorf("dns-a must judge the public record: %+v", byID["dns-a"])
	}
	if !strings.Contains(byID["router-25565"].Message, "203.0.113.7") {
		t.Errorf("the router test must use the public address: %+v", byID["router-25565"])
	}
}
