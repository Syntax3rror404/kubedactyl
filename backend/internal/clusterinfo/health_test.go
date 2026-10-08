package clusterinfo

import (
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/checks"
	"app/internal/settings"
)

func node(name string, ready bool) corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status}}},
	}
}

func healthyInputs() healthInputs {
	return healthInputs{
		crdResources: []string{"eggs", "gameservers", "users", "panelsettings", "invites", "gameservers/status"},
		permissions:  []Permission{{Label: "Create pods", Allowed: true}},
		nodes:        []corev1.Node{node("w1", true)},
		metrics:      true,
		cilium:       true,
		settings: v1alpha1.PanelSettingsSpec{
			StorageClasses:    []string{"longhorn"},
			LoadBalancerPools: []string{"general-pool"},
			OIDC:              v1alpha1.OIDCSettings{Enabled: true},
		},
		storageClasses: []settings.StorageClass{{Name: "longhorn"}},
		pools:          []settings.Pool{{Name: "general-pool", IPsAvailable: 55}},
		guarded:        true,
		policy:         "kubedactyl-panel",
	}
}

func byID(list []checks.Check) map[string]checks.Check {
	out := map[string]checks.Check{}
	for _, c := range list {
		out[c.ID] = c
	}
	return out
}

func TestHealthAllGood(t *testing.T) {
	for _, c := range evaluate(healthyInputs()) {
		if c.Status != checks.OK {
			t.Errorf("%s = %s (%s)", c.ID, c.Status, c.Message)
		}
	}
}

func TestHealthProblems(t *testing.T) {
	cases := []struct {
		name   string
		change func(*healthInputs)
		id     string
		status checks.Status
		msg    string
	}{
		{
			"CRD deleted",
			func(in *healthInputs) { in.crdResources = []string{"eggs", "users"} },
			"crds",
			checks.Error,
			"gameservers, panelsettings, invites",
		},
		{
			"permission missing",
			func(in *healthInputs) { in.permissions = append(in.permissions, Permission{Label: "Exec into pods"}) },
			"permissions",
			checks.Error,
			"Exec into pods",
		},
		{
			"node not ready",
			func(in *healthInputs) { in.nodes = append(in.nodes, node("m2", false)) },
			"nodes",
			checks.Warning,
			"1 of 2 not ready: m2",
		},
		{
			"no metrics-server",
			func(in *healthInputs) { in.metrics = false },
			"metrics",
			checks.Warning,
			"no CPU and memory",
		},
		{"no Cilium", func(in *healthInputs) { in.cilium = false }, "loadBalancer", checks.Error, "not installed"},
		{
			"enabled pool deleted",
			func(in *healthInputs) { in.pools = nil },
			"loadBalancer",
			checks.Error,
			"general-pool",
		},
		{
			"pool full",
			func(in *healthInputs) { in.pools[0].IPsAvailable = 0 },
			"loadBalancer",
			checks.Warning,
			"no free address",
		},
		{
			"IPv4 of a mixed pool full",
			func(in *healthInputs) {
				in.pools[0].Families = []settings.PoolFamily{
					{Family: "IPv4", Total: 2, Used: 2}, {Family: "IPv6", Total: 1.8e19, Available: 1.8e19},
				}
			},
			"loadBalancer",
			checks.Warning,
			"general-pool (IPv4)",
		},
		{
			"no pool enabled",
			func(in *healthInputs) { in.settings.LoadBalancerPools = nil },
			"loadBalancer",
			checks.Warning,
			"no pool",
		},
		{
			"storage class deleted",
			func(in *healthInputs) { in.storageClasses = nil },
			"storage",
			checks.Error,
			"longhorn",
		},
		{
			"no admission policy API",
			func(in *healthInputs) { in.policy = "" },
			"admission-policy",
			checks.Warning,
			"not limited",
		},
		{
			"admission policy deleted",
			func(in *healthInputs) { in.policyErr = errors.New("not found") },
			"admission-policy",
			checks.Error,
			"missing",
		},
		{
			"local panel",
			func(in *healthInputs) { in.guarded = false },
			"admission-policy",
			checks.Skipped,
			"kubeconfig",
		},
		{
			"single sign-on off",
			func(in *healthInputs) { in.settings.OIDC.Enabled = false },
			"oidc",
			checks.Skipped,
			"off",
		},
		{
			"identity provider down",
			func(in *healthInputs) { in.oidcErr = errors.New("discovery failed") },
			"oidc",
			checks.Error,
			"discovery failed",
		},
		{
			"no storage class enabled",
			func(in *healthInputs) { in.settings.StorageClasses = nil },
			"storage",
			checks.Error,
			"cannot be created",
		},
	}
	for _, c := range cases {
		in := healthyInputs()
		c.change(&in)
		got := byID(evaluate(in))[c.id]
		if got.Status != c.status || !strings.Contains(got.Message, c.msg) {
			t.Errorf("%s: %s = %s %q, want %s containing %q", c.name, c.id, got.Status, got.Message, c.status, c.msg)
		}
	}
}
