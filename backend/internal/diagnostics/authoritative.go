package diagnostics

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/netip"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// dnsPort is the port name servers listen on (a variable for tests).
var dnsPort uint16 = 53

// authoritative finds the name servers of the domain's zone the way a recursive resolver does:
// it starts at the servers of the top-level domain and follows the delegations down. A local
// DNS server that overrides the domain (a local zone) therefore cannot hide them. It returns a
// resolver that asks these servers directly, or nil when they cannot be reached.
func authoritative(ctx context.Context, domain string) Resolver {
	labels := strings.Split(strings.TrimSuffix(domain, "."), ".")
	if len(labels) < 2 {
		return nil
	}
	servers := topLevelServers(ctx, labels[len(labels)-1])
	for i := len(labels) - 2; i >= 0 && len(servers) > 0; i-- {
		next, err := delegation(ctx, servers, strings.Join(labels[i:], "."))
		if err != nil {
			return nil
		}
		if len(next) == 0 {
			break // the current servers answer for this name themselves
		}
		servers = next
	}
	if len(servers) == 0 {
		return nil
	}
	server := netip.AddrPortFrom(servers[0], dnsPort).String()
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server)
	}}
}

// topLevelServers are the name servers of a top-level domain ("de"), asked at the local DNS.
func topLevelServers(ctx context.Context, tld string) []netip.Addr {
	ns, err := net.DefaultResolver.LookupNS(ctx, tld)
	if err != nil {
		return nil
	}
	names := make([]string, len(ns))
	for i, n := range ns {
		names[i] = n.Host
	}
	return addresses(ctx, names, nil)
}

// delegation asks the servers for the name servers of zone. It returns their addresses, none
// when the servers answer for zone themselves, or an error when no server answered.
func delegation(ctx context.Context, servers []netip.Addr, zone string) ([]netip.Addr, error) {
	var lastErr error
	for _, server := range servers[:min(len(servers), 3)] {
		reply, err := exchange(ctx, server, zone)
		if err != nil {
			lastErr = err
			continue
		}
		names, glue := parseDelegation(reply, zone)
		return addresses(ctx, names, glue), nil
	}
	return nil, lastErr
}

// exchange sends a non-recursive NS query for zone over UDP and returns the reply.
func exchange(ctx context.Context, server netip.Addr, zone string) ([]byte, error) {
	name, err := dnsmessage.NewName(zone + ".")
	if err != nil {
		return nil, err
	}
	id := uint16(rand.N(1 << 16))
	query, err := (&dnsmessage.Message{
		Header:    dnsmessage.Header{ID: id},
		Questions: []dnsmessage.Question{{Name: name, Type: dnsmessage.TypeNS, Class: dnsmessage.ClassINET}},
	}).Pack()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "udp", netip.AddrPortFrom(server, dnsPort).String())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	if _, err := conn.Write(query); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	var p dnsmessage.Parser
	if h, err := p.Start(buf[:n]); err != nil || !h.Response || h.ID != id {
		return nil, errors.New("invalid reply from " + server.String())
	}
	return buf[:n], nil
}

// parseDelegation returns the name servers of zone from a reply (answer or referral) and the
// addresses the reply includes for them (glue).
func parseDelegation(reply []byte, zone string) (names []string, glue map[string]netip.Addr) {
	var msg dnsmessage.Message
	if err := msg.Unpack(reply); err != nil {
		return nil, nil
	}
	glue = map[string]netip.Addr{}
	want := strings.ToLower(strings.TrimSuffix(zone, ".") + ".")
	for _, rr := range append(msg.Answers, msg.Authorities...) {
		if ns, ok := rr.Body.(*dnsmessage.NSResource); ok && strings.ToLower(rr.Header.Name.String()) == want {
			names = append(names, ns.NS.String())
		}
	}
	for _, rr := range msg.Additionals {
		if a, ok := rr.Body.(*dnsmessage.AResource); ok {
			glue[strings.ToLower(rr.Header.Name.String())] = netip.AddrFrom4(a.A)
		}
	}
	return names, glue
}

// addresses resolves name server names to IPv4 addresses, preferring the glue of the reply.
func addresses(ctx context.Context, names []string, glue map[string]netip.Addr) []netip.Addr {
	var out []netip.Addr
	for _, n := range names {
		if ip, ok := glue[strings.ToLower(n)]; ok {
			out = append(out, ip)
			continue
		}
		if ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", strings.TrimSuffix(n, ".")); err == nil &&
			len(ips) > 0 {
			out = append(out, ips[0])
		}
	}
	return out
}
