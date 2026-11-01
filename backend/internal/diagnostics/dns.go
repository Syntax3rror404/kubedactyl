package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	"app/internal/checks"
)

// records are the addresses of the domain as one resolver sees them.
type records struct {
	v4, v6 []netip.Addr
	// public is false when the public view gave no address and the local answer is used.
	public bool
	// err is set when the IPv4 addresses could not be looked up (not when there are none).
	err error
}

// lookupPublic resolves the domain as players on the internet see it.
//  1. Public: the domain's name servers, asked directly.
//  2. Local: the DNS of the server's network (marked in the records), when the public view gives
//     no address: its name servers cannot be reached, fail, or answer only with a CNAME into a
//     zone they do not serve.
func (d *Diagnoser) lookupPublic(ctx context.Context, domain string) records {
	// A trailing dot keeps the resolver from trying the search domains of the pod first.
	fqdn := strings.TrimSuffix(domain, ".") + "."
	// 1. Public
	if pub := d.resolvePublic(ctx, fqdn); len(pub.v4)+len(pub.v6) > 0 {
		pub.public = true
		return pub
	}
	// 2. Local
	var local Resolver = net.DefaultResolver
	if d.Resolver != nil {
		local = d.Resolver
	}
	ctx, cancel := context.WithTimeout(ctx, d.timeout())
	defer cancel()
	return resolve(ctx, local, fqdn)
}

// resolvePublic asks the public name servers of the domain; no records when they are not found.
func (d *Diagnoser) resolvePublic(ctx context.Context, fqdn string) records {
	ctx, cancel := context.WithTimeout(ctx, 2*d.timeout())
	defer cancel()
	public := d.PublicResolver
	if public == nil {
		public = authoritative(ctx, fqdn)
	}
	if public == nil {
		return records{}
	}
	return resolve(ctx, public, fqdn)
}

// resolve asks for both address families at once.
func resolve(ctx context.Context, r Resolver, domain string) records {
	var out records
	var err error
	var wg sync.WaitGroup
	wg.Go(func() { out.v4, err = r.LookupNetIP(ctx, "ip4", domain) })
	out.v6, _ = r.LookupNetIP(ctx, "ip6", domain)
	wg.Wait()
	if !notFound(err) {
		out.err = err
	}
	return out
}

// notFound reports a lookup that was answered: the name or its records do not exist.
func notFound(err error) bool {
	var dnsErr *net.DNSError
	return err == nil || errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

func aRecord(in Input, pub records) checks.Check {
	const id, label = "dns-a", "DNS A record (IPv4)"
	v4 := pub.v4
	if len(v4) == 0 && pub.err != nil {
		return checks.New(id, label, checks.Warning, fmt.Sprintf("The DNS records of %s could not be read. "+
			"Try again in a moment.", in.Domain))
	}
	if len(v4) == 0 {
		return checks.New(id, label, checks.Error, fmt.Sprintf("%s has no A record, so players cannot find the "+
			"server. An administrator must create one that points to the public IP of the server's network.",
			in.Domain))
	}
	lb, _ := netip.ParseAddr(in.Server.Status.Address)
	const repoint = "an administrator must point the record to the public IP"
	var parts []string
	status := checks.OK
	for _, ip := range v4 {
		switch {
		case ip == lb:
			parts = append(parts, shown(in, ip.String(), "the load balancer")+
				": the server's internal address, which players on the internet cannot reach; "+repoint)
			status = checks.Error
		case internal(ip):
			parts = append(parts, shown(in, ip.String(), "an internal address")+
				": an internal address players on the internet cannot reach; "+repoint)
			status = checks.Error
		default:
			parts = append(parts, ip.String()+": a public address, players find the server")
		}
	}
	msg := strings.Join(parts, "; ") + "."
	if !pub.public {
		msg += " (Answer of the DNS in the server's network; the public name servers gave no address.)"
	}
	return checks.New(id, label, status, msg)
}

// shown is the address for administrators and the hidden text for users, who never see internal addresses.
func shown(in Input, ip, hidden string) string {
	if in.ShowAddresses {
		return ip
	}
	return hidden
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
	return checks.New(id, label, checks.Warning, fmt.Sprintf("%s also points to %s. Players with IPv6 try this "+
		"address first. If they cannot connect, an administrator must allow the ports for IPv6 too or remove the "+
		"AAAA record.", domain, strings.Join(ips, ", ")))
}

func firstPublic(ips []netip.Addr) netip.Addr {
	for _, ip := range ips {
		if !internal(ip) {
			return ip
		}
	}
	return netip.Addr{}
}

// sharedAddressSpace is the carrier-grade NAT range (RFC 6598), not reachable from the internet.
var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

// internal reports addresses players on the internet cannot reach.
func internal(ip netip.Addr) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || sharedAddressSpace.Contains(ip)
}
