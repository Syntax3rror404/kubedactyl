package diagnostics

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"app/internal/checks"
)

// records are the addresses of the domain as one resolver sees them.
type records struct {
	v4, v6 []netip.Addr
	// public is false when the public view could not be asked and the local answer is used.
	public bool
}

// lookupPublic resolves the domain as players on the internet see it. Without access to the
// public name servers it falls back to the DNS of the server's network (marked in the records).
func (d *Diagnoser) lookupPublic(ctx context.Context, domain string) records {
	ctx, cancel := context.WithTimeout(ctx, 2*d.timeout())
	defer cancel()
	publicResolver := d.PublicResolver
	if publicResolver == nil {
		publicResolver = authoritative(ctx, domain)
	}
	if publicResolver != nil {
		pub := resolve(ctx, publicResolver, domain)
		pub.public = true
		return pub
	}
	var local Resolver = net.DefaultResolver
	if d.Resolver != nil {
		local = d.Resolver
	}
	return resolve(ctx, local, domain)
}

func resolve(ctx context.Context, r Resolver, domain string) records {
	var out records
	out.v4, _ = r.LookupNetIP(ctx, "ip4", domain)
	out.v6, _ = r.LookupNetIP(ctx, "ip6", domain)
	return out
}

func aRecord(in Input, pub records) checks.Check {
	const id, label = "dns-a", "DNS A record (IPv4)"
	v4 := pub.v4
	if len(v4) == 0 {
		return checks.New(
			id,
			label,
			checks.Error,
			fmt.Sprintf(
				"%s has no A record, so players cannot find the server. An administrator must create one "+
					"that points to the public IP of the server's network.",
				in.Domain,
			),
		)
	}
	lb, _ := netip.ParseAddr(in.Server.Status.Address)
	var parts []string
	status := checks.OK
	for _, ip := range v4 {
		switch {
		case ip == lb:
			parts = append(
				parts,
				describe(
					in,
					ip,
				)+": the server's internal address, which players on the internet cannot reach; "+
					"an administrator must point the record to the public IP",
			)
			status = checks.Error
		case internal(ip):
			parts = append(
				parts,
				private(
					in,
					ip,
				)+": an internal address players on the internet cannot reach; "+
					"an administrator must point the record to the public IP",
			)
			status = checks.Error
		default:
			parts = append(parts, ip.String()+": a public address, players find the server")
		}
	}
	msg := strings.Join(parts, "; ") + "."
	if !pub.public {
		msg += " (Answer of the DNS in the server's network; the public name servers could not be asked.)"
	}
	return checks.New(id, label, status, msg)
}

// private shows an internal address only to administrators.
func private(in Input, ip netip.Addr) string {
	if in.ShowAddresses {
		return ip.String()
	}
	return "an internal address"
}

func aaaaRecord(domain string, v6 []netip.Addr) checks.Check {
	const id, label = "dns-aaaa", "DNS AAAA record (IPv6)"
	if len(v6) == 0 {
		return checks.New(id, label, checks.OK, "No AAAA record. Players connect via IPv4.")
	}
	ips := make([]string, len(v6))
	for i, ip := range v6 {
		ips[i] = ip.String()
	}
	return checks.New(
		id,
		label,
		checks.Warning,
		fmt.Sprintf(
			"%s also points to %s. Players with IPv6 try this address first. If they cannot connect, "+
				"an administrator must allow the ports for IPv6 too or remove the AAAA record.",
			domain,
			strings.Join(ips, ", "),
		),
	)
}

func firstPublic(ips []netip.Addr) netip.Addr {
	for _, ip := range ips {
		if !internal(ip) {
			return ip
		}
	}
	return netip.Addr{}
}

// internal reports addresses players on the internet cannot reach.
func internal(ip netip.Addr) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

func describe(in Input, ip netip.Addr) string {
	if in.ShowAddresses {
		return ip.String()
	}
	return "the load balancer"
}
