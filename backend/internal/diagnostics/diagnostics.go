// Package diagnostics checks why players cannot reach a game server: the typical, simple causes
// from the installation over the load balancer address to the public DNS records and the router
// forwarding. It is written for server owners, who connect from the internet like their players:
// messages describe what players see and say when an administrator has to act. The panel itself
// tests from the server's network; messages say so where it matters.
package diagnostics

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"app/api/v1alpha1"
	"app/internal/checks"
)

// Resolver looks up addresses (net.Resolver implements it).
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// Diagnoser runs the checks.
//
// Resolver answers like the DNS of the panel's network (default: the system resolver). Local
// DNS servers often override the own domain with an internal address, so the records players
// on the internet see come from PublicResolver: by default the domain's authoritative name
// servers, asked directly (no third-party resolver). Dial defaults to a TCP dialer.
type Diagnoser struct {
	Resolver       Resolver
	PublicResolver Resolver
	Dial           func(ctx context.Context, network, address string) (net.Conn, error)
	Timeout        time.Duration
}

// Input is the server to check.
type Input struct {
	Server *v1alpha1.GameServer
	// Domain is the external domain of the panel settings ("" = players see the IP).
	Domain string
	// ShowAddresses shows the load balancer IP (administrators); users only see the domain.
	ShowAddresses bool
}

// Run returns the checks in the order a player's connection passes them.
func (d *Diagnoser) Run(ctx context.Context, in Input) []checks.Check {
	gs := in.Server
	list := []checks.Check{installation(gs), state(gs), restart(gs), disk(gs), address(in)}
	if in.Domain == "" {
		return append(list,
			checks.New("domain", "External domain", checks.Skipped,
				"No external domain is set. Players connect to the server's IP address directly."))
	}
	list = append(list, checks.New("domain", "External domain", checks.OK,
		fmt.Sprintf("Players connect to %s.", in.Domain)))
	pub := d.lookupPublic(ctx, in.Domain)
	list = append(list, aRecord(in, pub), aaaaRecord(in.Domain, pub.v6))
	return append(list, d.routerChecks(ctx, in, pub.v4)...)
}

func installation(gs *v1alpha1.GameServer) checks.Check {
	const id, label = "install", "Installation"
	switch gs.Status.Phase {
	case v1alpha1.PhaseInstallFailed:
		return checks.New(id, label, checks.Error, "The install script failed"+exitCode(gs.Status.InstallExitCode)+
			". See the console output and reinstall it (Settings → Danger zone).")
	case v1alpha1.PhaseInstalling, v1alpha1.PhasePending:
		return checks.New(id, label, checks.Warning, "The server is still being installed.")
	}
	return checks.New(id, label, checks.OK, "Installed.")
}

func state(gs *v1alpha1.GameServer) checks.Check {
	const id, label = "state", "Server state"
	switch {
	case gs.Spec.Suspended:
		return checks.New(id, label, checks.Error, "The server is suspended by an administrator.")
	case gs.Status.Phase == v1alpha1.PhaseRunning:
		return checks.New(id, label, checks.OK, "The server is running.")
	case gs.Status.Phase == v1alpha1.PhaseStarting:
		return checks.New(
			id, label, checks.Warning,
			"The server is starting. It counts as running once the egg's startup line appears.",
		)
	case gs.Status.LastExitCode != nil && *gs.Status.LastExitCode == 137:
		return checks.New(
			id,
			label,
			checks.Error,
			"The server was killed (exit code 137). Usually it ran out of memory. "+
				"An administrator can give it more memory.",
		)
	case gs.Status.LastExitCode != nil && *gs.Status.LastExitCode != 0 && gs.Spec.State == v1alpha1.PowerRunning:
		return checks.New(
			id, label, checks.Error, "The server crashed"+exitCode(gs.Status.LastExitCode)+". See the console output.",
		)
	}
	return checks.New(id, label, checks.Warning, "The server is stopped. Start it so players can connect.")
}

func restart(gs *v1alpha1.GameServer) checks.Check {
	const id, label = "restart", "Settings applied"
	if gs.Status.RestartRequired {
		return checks.New(
			id, label, checks.Warning, "Settings changed since the start: restart the server to apply them.",
		)
	}
	return checks.New(id, label, checks.OK, "The running server uses the current settings.")
}

func disk(gs *v1alpha1.GameServer) checks.Check {
	const id, label = "disk", "Disk space"
	if gs.Status.DiskUsedBytes == nil || gs.Spec.Resources.DiskMiB == 0 {
		return checks.New(id, label, checks.Skipped, "Not measured yet. Open the files once.")
	}
	used := float64(*gs.Status.DiskUsedBytes) / float64(gs.Spec.Resources.DiskMiB<<20) * 100
	msg := fmt.Sprintf("%.0f %% of %d MiB used.", used, gs.Spec.Resources.DiskMiB)
	switch {
	case used >= 98:
		return checks.New(
			id,
			label,
			checks.Error,
			msg+" The disk is full: the game cannot save. Delete files, or ask an administrator for more space.",
		)
	case used >= 90:
		return checks.New(
			id, label, checks.Warning, msg+" Almost full. Delete files soon, or ask an administrator for more space.",
		)
	}
	return checks.New(id, label, checks.OK, msg)
}

func address(in Input) checks.Check {
	const id, label = "address", "Server address"
	gs := in.Server
	if gs.Status.Address == "" {
		return checks.New(
			id,
			label,
			checks.Error,
			fmt.Sprintf(
				"The server has no address yet (pool %q). An administrator must check the pool: "+
					"it may have no free address left.",
				gs.Spec.LoadBalancerPool,
			),
		)
	}
	if !in.ShowAddresses {
		return checks.New(id, label, checks.OK, "The server has an address.")
	}
	return checks.New(
		id, label, checks.OK, fmt.Sprintf("%s from the pool %q.", gs.Status.Address, gs.Spec.LoadBalancerPool),
	)
}

func exitCode(code *int32) string {
	if code == nil {
		return ""
	}
	return " (exit code " + strconv.Itoa(int(*code)) + ")"
}
