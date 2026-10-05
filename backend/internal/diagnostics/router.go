package diagnostics

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"app/api/v1alpha1"
	"app/internal/checks"
)

// routerChecks tests each port through the public address (NAT loopback) while the server runs.
func (d *Diagnoser) routerChecks(ctx context.Context, in Input, v4 []netip.Addr) []checks.Check {
	gs := in.Server
	public := firstPublic(v4)
	out := make([]checks.Check, len(gs.Spec.Ports))
	var wg sync.WaitGroup
	for i, port := range gs.Spec.Ports {
		id, label := fmt.Sprintf("router-%d", port), fmt.Sprintf("Router forwarding, port %d", port)
		switch {
		case !public.IsValid():
			out[i] = checks.New(
				id, label, checks.Skipped,
				"The domain has no public IPv4 address, so there is no router forwarding to test.",
			)
		case gs.Status.Phase != v1alpha1.PhaseRunning:
			out[i] = checks.New(id, label, checks.Skipped, "Start the server to test the forwarding.")
		default:
			wg.Go(func() { out[i] = d.routerCheck(ctx, id, label, in, netip.AddrPortFrom(public, uint16(port))) })
		}
	}
	wg.Wait()
	return out
}

// routerCheck tests one port through the public address, but only for games that accept TCP
// on it: the panel cannot test UDP, and many games (e.g. Factorio) only use UDP. So it first
// connects to the server's own address; a game without TCP gets a neutral note instead.
func (d *Diagnoser) routerCheck(ctx context.Context, id, label string, in Input, target netip.AddrPort) checks.Check {
	lb := "the server's address"
	if in.ShowAddresses {
		lb = in.Server.Status.Address
	}
	inside, err := netip.ParseAddr(in.Server.Status.Address)
	if err != nil || d.tcpOpen(ctx, netip.AddrPortFrom(inside, target.Port())) != nil {
		return checks.New(
			id,
			label,
			checks.Skipped,
			fmt.Sprintf(
				"The game does not accept TCP on port %d (it probably uses only UDP), so the forwarding "+
					"cannot be tested (UDP cannot be tested from the panel).",
				target.Port(),
			),
		)
	}
	if err := d.tcpOpen(ctx, target); err != nil {
		return checks.New(
			id,
			label,
			checks.Warning,
			fmt.Sprintf(
				"The game accepts TCP on port %d, but not through the public address %s. "+
					"Either the router does not forward the port to %s, or it does not answer on its own public "+
					"address from inside (then players outside can still connect; ask one to try).",
				target.Port(),
				target.Addr(),
				lb,
			),
		)
	}
	return checks.New(
		id, label, checks.OK,
		fmt.Sprintf("TCP port %d answers through the public address %s.", target.Port(), target.Addr()),
	)
}

// tcpOpen reports whether a TCP connection to target can be opened.
func (d *Diagnoser) tcpOpen(ctx context.Context, target netip.AddrPort) error {
	dial := d.Dial
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	ctx, cancel := context.WithTimeout(ctx, d.timeout())
	defer cancel()
	conn, err := dial(ctx, "tcp", target.String())
	if err != nil {
		return err
	}
	return conn.Close()
}

func (d *Diagnoser) timeout() time.Duration {
	if d.Timeout > 0 {
		return d.Timeout
	}
	return 3 * time.Second
}
