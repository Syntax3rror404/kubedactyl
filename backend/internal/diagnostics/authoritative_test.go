package diagnostics

import (
	"context"
	"net"
	"net/netip"
	"os"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

// dnsServer answers UDP queries on 127.0.0.1 with reply.
func dnsServer(t *testing.T, reply func(q dnsmessage.Message) dnsmessage.Message) netip.AddrPort {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			var q dnsmessage.Message
			if q.Unpack(buf[:n]) != nil || len(q.Questions) == 0 {
				continue
			}
			r := reply(q)
			r.Header.ID, r.Header.Response, r.Questions = q.Header.ID, true, q.Questions
			out, _ := r.Pack()
			_, _ = conn.WriteTo(out, addr)
		}
	}()
	return conn.LocalAddr().(*net.UDPAddr).AddrPort()
}

// fakeNameServer answers NS queries like a parent zone: a referral to ns1.example.net with glue.
func fakeNameServer(t *testing.T) netip.Addr {
	t.Helper()
	server := dnsServer(t, func(q dnsmessage.Message) dnsmessage.Message {
		ns := dnsmessage.MustNewName("ns1.example.net.")
		return dnsmessage.Message{
			Authorities: []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{
					Name: q.Questions[0].Name, Type: dnsmessage.TypeNS, Class: dnsmessage.ClassINET,
				},
				Body: &dnsmessage.NSResource{NS: ns},
			}},
			Additionals: []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{Name: ns, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET},
				Body:   &dnsmessage.AResource{A: [4]byte{203, 0, 113, 53}},
			}},
		}
	})
	old := dnsPort
	dnsPort = server.Port()
	t.Cleanup(func() { dnsPort = old })
	return server.Addr()
}

// TestNameServersTryTheNext: a failing name server is skipped, the next one answers.
func TestNameServersTryTheNext(t *testing.T) {
	failing := dnsServer(t, func(dnsmessage.Message) dnsmessage.Message {
		return dnsmessage.Message{Header: dnsmessage.Header{RCode: dnsmessage.RCodeServerFailure}}
	})
	answering := dnsServer(t, func(q dnsmessage.Message) dnsmessage.Message {
		r := dnsmessage.Message{Header: dnsmessage.Header{Authoritative: true}}
		if q.Questions[0].Type == dnsmessage.TypeA {
			r.Answers = []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{Name: q.Questions[0].Name, Type: dnsmessage.TypeA,
					Class: dnsmessage.ClassINET, TTL: 60},
				Body: &dnsmessage.AResource{A: [4]byte{203, 0, 113, 7}},
			}}
		}
		return r
	})
	ns := nameServers{failing, answering}
	got, err := ns.LookupNetIP(context.Background(), "ip4", "play.example.com.")
	if err != nil || len(got) != 1 || got[0] != netip.MustParseAddr("203.0.113.7") {
		t.Errorf("ip4 = %v, %v", got, err)
	}
	// A name without records of the family is an answer, not a failure.
	if _, err := ns.LookupNetIP(context.Background(), "ip6", "play.example.com."); !notFound(err) || err == nil {
		t.Errorf("ip6 = %v, want not found", err)
	}
	if _, err := (nameServers{failing}).LookupNetIP(context.Background(), "ip4", "play.example.com."); notFound(err) {
		t.Errorf("failing server = %v, want an error", err)
	}
}

func TestDelegationFollowsReferral(t *testing.T) {
	server := fakeNameServer(t)
	got, err := delegation(context.Background(), []netip.Addr{server}, "example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != netip.MustParseAddr("203.0.113.53") {
		t.Errorf("delegation = %v, want the glue address of ns1.example.net", got)
	}
}

func TestParseDelegationIgnoresOtherZones(t *testing.T) {
	reply, _ := (&dnsmessage.Message{
		Header: dnsmessage.Header{Response: true},
		Authorities: []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{
				Name: dnsmessage.MustNewName("other.org."), Type: dnsmessage.TypeNS, Class: dnsmessage.ClassINET,
			},
			Body: &dnsmessage.NSResource{NS: dnsmessage.MustNewName("ns.other.org.")},
		}},
	}).Pack()
	if names, _ := parseDelegation(reply, "example.org"); len(names) != 0 {
		t.Errorf("names for another zone must be ignored: %v", names)
	}
}

// DIAG_IT_DOMAIN=example.com go test ./internal/diagnostics -run Integration -v asks the real
// name servers (needs internet). Run it where a local DNS server overrides the domain.
func TestAuthoritativeIntegration(t *testing.T) {
	domain := os.Getenv("DIAG_IT_DOMAIN")
	if domain == "" {
		t.Skip("set DIAG_IT_DOMAIN")
	}
	r := authoritative(context.Background(), domain)
	if r == nil {
		t.Fatal("authoritative name servers not found")
	}
	ips, err := r.LookupNetIP(context.Background(), "ip4", domain)
	local, _ := net.DefaultResolver.LookupNetIP(context.Background(), "ip4", domain)
	t.Logf("public (authoritative): %v %v, local: %v", ips, err, local)
	if err != nil || len(ips) == 0 {
		t.Fatal("no public A record")
	}
}
