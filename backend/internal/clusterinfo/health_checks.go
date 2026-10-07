package clusterinfo

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"app/internal/checks"
	"app/internal/settings"
)

// checkCRDs: the panel's own resources must be served.
func checkCRDs(in healthInputs) checks.Check {
	const id, label = "crds", "Kubedactyl CRDs"
	var missing []string
	for _, r := range []string{"eggs", "gameservers", "users", "panelsettings", "invites"} {
		if !slices.Contains(in.crdResources, r) {
			missing = append(missing, r)
		}
	}
	switch {
	case in.crdErr != nil:
		return checks.New(
			id, label, checks.Error, "kubedactyl.io/v1alpha1 is not served, restart the panel to install the CRDs",
		)
	case len(missing) > 0:
		return checks.New(
			id, label, checks.Error, "missing: "+strings.Join(missing, ", ")+", restart the panel to install them",
		)
	}
	return checks.New(id, label, checks.OK, "")
}

// checkPermissions: every permission check of the service account must pass.
func checkPermissions(in healthInputs) checks.Check {
	var denied []string
	for _, p := range in.permissions {
		if !p.Allowed {
			denied = append(denied, p.Label)
		}
	}
	if len(denied) > 0 {
		return checks.New("permissions", "Permissions", checks.Error, "not allowed: "+strings.Join(denied, ", "))
	}
	return checks.New("permissions", "Permissions", checks.OK, "")
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

func nodeReady(n corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

// checkMetrics: without metrics-server there is no CPU and memory usage.
func checkMetrics(in healthInputs) checks.Check {
	if !in.metrics {
		return checks.New(
			"metrics", "Metrics server", checks.Warning, "metrics.k8s.io is not available, no CPU and memory usage",
		)
	}
	return checks.New("metrics", "Metrics server", checks.OK, "")
}

// checkLoadBalancer: Cilium LB IPAM is installed and every enabled pool exists and has free IPs.
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

// checkStorage: at least one storage class is enabled and all enabled ones exist.
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
	var gone []string
	for _, name := range in.settings.StorageClasses {
		if !slices.ContainsFunc(in.storageClasses, func(c settings.StorageClass) bool { return c.Name == name }) {
			gone = append(gone, name)
		}
	}
	if len(gone) > 0 {
		return checks.New(id, label, checks.Error, "enabled but missing in the cluster: "+strings.Join(gone, ", "))
	}
	return checks.New(id, label, checks.OK, "")
}

// checkAdmissionPolicy: in the cluster, the admission policy must limit the panel's cluster
// wide permissions (namespaces, volumes, role bindings) to its own namespaces.
func checkAdmissionPolicy(in healthInputs) checks.Check {
	const id, label = "admission-policy", "Permission guard"
	switch {
	case !in.guarded:
		return checks.New(id, label, checks.Skipped, "the panel runs with a kubeconfig, not its service account")
	case in.policy == "":
		return checks.New(
			id,
			label,
			checks.Warning,
			"the cluster has no ValidatingAdmissionPolicy (Kubernetes 1.30+), "+
				"the panel's cluster wide permissions are not limited to its own namespaces",
		)
	case in.policyErr != nil:
		return checks.New(
			id, label, checks.Error,
			"admission policy "+in.policy+" is missing, upgrade the chart: "+in.policyErr.Error(),
		)
	}
	return checks.New(id, label, checks.OK, "")
}

// checkOIDC: while single sign-on is on, its identity provider must answer.
func checkOIDC(in healthInputs) checks.Check {
	const id, label = "oidc", "Single sign-on"
	switch {
	case !in.settings.OIDC.Enabled:
		return checks.New(id, label, checks.Skipped, "single sign-on is off")
	case in.oidcErr != nil:
		return checks.New(id, label, checks.Error, "users cannot sign in through it: "+in.oidcErr.Error())
	}
	return checks.New(id, label, checks.OK, "")
}
