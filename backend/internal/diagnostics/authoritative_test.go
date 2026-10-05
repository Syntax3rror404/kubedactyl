package diagnostics

import (
	"context"
	"net"
	"net/netip"
	"os"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

// fakeNameServer answers NS queries like a parent zone: a referral to ns1.example.net with glue.
func fakeNameServer(t *testing.T) netip.Addr {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	old := dnsPort
	dnsPort = conn.LocalAddr().(*net.UDPAddr).AddrPort().Port()
	t.Cleanup(func() { dnsPort = old })
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
			ns := dnsmessage.MustNewName("ns1.example.net.")
			reply := dnsmessage.Message{
				Header:    dnsmessage.Header{ID: q.Header.ID, Response: true},
				Questions: q.Questions,
				Authorities: []dnsmessage.Resource{
					{
						Header: dnsmessage.ResourceHeader{
							Name:  q.Questions[0].Name,
							Type:  dnsmessage.TypeNS,
							Class: dnsmessage.ClassINET,
						},
						Body: &dnsmessage.NSResource{NS: ns},
					},
				},
				Additionals: []dnsmessage.Resource{{
					Header: dnsmessage.ResourceHeader{Name: ns, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET},
					Body:   &dnsmessage.AResource{A: [4]byte{203, 0, 113, 53}},
				}},
			}
			out, _ := reply.Pack()
			_, _ = conn.WriteTo(out, addr)
		}
	}()
	return netip.MustParseAddr("127.0.0.1")
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
// name servers (needs internet); used to verify the lookup inside a cluster with a local DNS override.
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
