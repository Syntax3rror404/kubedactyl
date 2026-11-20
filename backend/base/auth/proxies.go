package auth

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Proxies are the reverse proxies (IPs and networks) whose X-Forwarded-For and X-Real-IP headers name the client.
// Without any, the client is the peer address, so nobody can fake their IP (login throttling).
type Proxies []netip.Prefix

// ParseProxies reads a comma separated list of IPs and CIDRs ("10.0.0.0/8, 192.168.1.10").
func ParseProxies(v string) (Proxies, error) {
	var out Proxies
	for s := range strings.SplitSeq(v, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			addr, aerr := netip.ParseAddr(s)
			if aerr != nil {
				return nil, fmt.Errorf("trusted proxy %q: not an IP or CIDR", s)
			}
			p = netip.PrefixFrom(addr.Unmap(), addr.Unmap().BitLen())
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

// trusts reports whether ip is one of the proxies.
func (p Proxies) trusts(ip netip.Addr) bool {
	for _, prefix := range p {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP is the IP of the client: the peer, or, when the peer is a trusted proxy, the last address in
// X-Forwarded-For (then X-Real-IP) that is not a trusted proxy itself.
func (p Proxies) ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !p.trusts(peer) {
		return peer.String()
	}
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if ip, ok := p.forwardedFor(r.Header.Get(header)); ok {
			return ip.String()
		}
	}
	return peer.String()
}

// forwardedFor walks the addresses of a forwarding header from the nearest proxy back to the client and returns
// the first one that is not a trusted proxy (or the first in the list). A malformed entry ends the walk: what
// stands before it cannot be trusted.
func (p Proxies) forwardedFor(header string) (netip.Addr, bool) {
	if header == "" {
		return netip.Addr{}, false
	}
	items := strings.Split(header, ",")
	for i := len(items) - 1; i >= 0; i-- {
		ip, err := netip.ParseAddr(strings.TrimSpace(items[i]))
		if err != nil {
			return netip.Addr{}, false
		}
		if ip = ip.Unmap(); i == 0 || !p.trusts(ip) {
			return ip, true
		}
	}
	return netip.Addr{}, false
}
