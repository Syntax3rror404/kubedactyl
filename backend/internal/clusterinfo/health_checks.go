package clusterinfo

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"app/internal/checks"
	"app/internal/settings"
)

// checkCRDs requires the panel's own resources to be served.
func checkCRDs(in healthInputs) checks.Check {
	const id, label = "crds", "Kubedactyl CRDs"
	absent := missing([]string{"eggs", "gameservers", "users", "panelsettings", "invites"}, func(r string) bool {
		return slices.Contains(in.crdResources, r)
	})
	switch {
	case in.crdErr != nil:
		return checks.New(
			id, label, checks.Error, "kubedactyl.io/v1alpha1 is not served, restart the panel to install the CRDs",
		)
	case len(absent) > 0:
		return checks.New(
			id, label, checks.Error, "missing: "+strings.Join(absent, ", ")+", restart the panel to install them",
		)
	}
	return checks.New(id, label, checks.OK, "")
}

// checkPermissions requires every permission check of the service account to pass.
func checkPermissions(in healthInputs) checks.Check {
	const id, label = "permissions", "Permissions"
	var denied []string
	for _, p := range in.permissions {
		if !p.Allowed {
			denied = append(denied, p.Label)
		}
	}
	if len(denied) > 0 {
		return checks.New(id, label, checks.Error, "not allowed: "+strings.Join(denied, ", "))
	}
	return checks.New(id, label, checks.OK, "")
}

// checkNodes warns about nodes that are not ready.
func checkNodes(in healthInputs) checks.Check {
	const id, label = "nodes", "Nodes"
	if in.nodesErr != nil {
		return checks.New(id, label, checks.Error, "cannot list nodes: "+in.nodesErr.Error())
	}
	var notReady []string
	for _, n := range in.nodes {
		if !nodeReady(n) {
			notReady = append(notReady, n.Name)
		}
	}
	if len(notReady) > 0 {
		return checks.New(
			id, label, checks.Warning,
			fmt.Sprintf("%d of %d not ready: %s", len(notReady), len(in.nodes), strings.Join(notReady, ", ")),
		)
	}
	return checks.New(id, label, checks.OK, "")
}

// missing lists the names the cluster does not have.
func missing(names []string, has func(string) bool) []string {
	var out []string
	for _, name := range names {
		if !has(name) {
			out = append(out, name)
		}
	}
	return out
}

func nodeReady(n corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

// checkMetrics warns without metrics-server: there is no CPU and memory usage then.
func checkMetrics(in healthInputs) checks.Check {
	const id, label = "metrics", "Metrics server"
	if !in.metrics {
		return checks.New(id, label, checks.Warning, "metrics.k8s.io is not available, no CPU and memory usage")
	}
	return checks.New(id, label, checks.OK, "")
}

// checkLoadBalancer requires Cilium LB IPAM and every enabled pool, with free IPs in each family.
func checkLoadBalancer(in healthInputs) checks.Check {
	const id, label = "loadBalancer", "Load balancer pools"
	switch {
	case !in.cilium:
		return checks.New(
			id, label, checks.Error,
			"Cilium LB IPAM (CiliumLoadBalancerIPPool) is not installed, servers get no address",
		)
	case in.settingsErr != nil || in.poolsErr != nil:
		return checks.New(id, label, checks.Error, "cannot read the pools: "+errText(in.settingsErr, in.poolsErr))
	case len(in.settings.LoadBalancerPools) == 0:
		return checks.New(id, label, checks.Warning, "no pool is enabled in the settings")
	}
	var gone, full []string
	for _, name := range in.settings.LoadBalancerPools {
		i := slices.IndexFunc(in.pools, func(p settings.Pool) bool { return p.Name == name })
		switch {
		case i < 0:
			gone = append(gone, name)
		case in.pools[i].IPsAvailable == 0:
			full = append(full, name)
		default:
			// A mixed pool can run out of IPv4 while it still has IPv6 addresses.
			for _, f := range in.pools[i].Families {
				if f.Available == 0 {
					full = append(full, name+" ("+f.Family+")")
				}
			}
		}
	}
	switch {
	case len(gone) > 0:
		return checks.New(id, label, checks.Error, "enabled but missing in the cluster: "+strings.Join(gone, ", "))
	case len(full) > 0:
		return checks.New(id, label, checks.Warning, "no free address: "+strings.Join(full, ", "))
	}
	return checks.New(id, label, checks.OK, "")
}

// checkStorage requires at least one enabled storage class, and every enabled one must exist.
func checkStorage(in healthInputs) checks.Check {
	const id, label = "storage", "Storage classes"
	switch {
	case in.settingsErr != nil || in.storageErr != nil:
		return checks.New(
			id, label, checks.Error, "cannot read the storage classes: "+errText(in.settingsErr, in.storageErr),
		)
	case len(in.settings.StorageClasses) == 0:
		return checks.New(id, label, checks.Error, "no storage class is enabled, servers cannot be created")
	}
	gone := missing(in.settings.StorageClasses, func(name string) bool {
		return slices.ContainsFunc(in.storageClasses, func(c settings.StorageClass) bool { return c.Name == name })
	})
	if len(gone) > 0 {
		return checks.New(id, label, checks.Error, "enabled but missing in the cluster: "+strings.Join(gone, ", "))
	}
	return checks.New(id, label, checks.OK, "")
}

// checkAdmissionPolicy requires, in the cluster, the admission policy that limits the panel's
// cluster wide permissions (namespaces, volumes, role bindings) to its own namespaces.
func checkAdmissionPolicy(in healthInputs) checks.Check {
	const id, label = "admission-policy", "Permission guard"
	switch {
	case !in.guarded:
		return checks.New(id, label, checks.Skipped, "the panel runs with a kubeconfig, not its service account")
	case in.policy == "":
		return checks.New(id, label, checks.Warning,
			"the cluster has no ValidatingAdmissionPolicy (Kubernetes 1.30+), "+
				"the panel's cluster wide permissions are not limited to its own namespaces")
	case in.policyErr != nil:
		return checks.New(
			id, label, checks.Error,
			"admission policy "+in.policy+" is missing, upgrade the chart: "+in.policyErr.Error(),
		)
	}
	return checks.New(id, label, checks.OK, "")
}

// checkOIDC requires the identity provider to answer while single sign-on is on.
func checkOIDC(in healthInputs) checks.Check {
	const id, label = "oidc", "OIDC"
	switch {
	case !in.settings.OIDC.Enabled:
		return checks.New(id, label, checks.Skipped, "OIDC is disabled")
	case in.oidcErr != nil:
		return checks.New(id, label, checks.Error, "users cannot sign in through it: "+in.oidcErr.Error())
	}
	return checks.New(id, label, checks.OK, "")
}
