package controller

import (
	"slices"
	"testing"
)

func TestIsolationPolicy(t *testing.T) {
	p := IsolationPolicy("kubedactyl-user-alice")
	if len(p.Spec.PolicyTypes) != 1 || p.Spec.PolicyTypes[0] != "Egress" {
		t.Fatalf("only egress may be restricted (players connect from outside): %v", p.Spec.PolicyTypes)
	}
	if len(p.Spec.PodSelector.MatchLabels) != 0 {
		t.Error("the policy must cover every pod of the namespace")
	}
	var sameNamespace, dns, internet bool
	for _, rule := range p.Spec.Egress {
		for _, to := range rule.To {
			switch {
			case to.IPBlock != nil && to.IPBlock.CIDR == "0.0.0.0/0":
				internet = slices.Contains(to.IPBlock.Except, "10.0.0.0/8") &&
					slices.Contains(to.IPBlock.Except, "192.168.0.0/16") &&
					slices.Contains(to.IPBlock.Except, "169.254.0.0/16")
			case to.NamespaceSelector != nil:
				dns = to.PodSelector.MatchLabels["k8s-app"] == "kube-dns" && len(rule.Ports) == 2
			case to.PodSelector != nil && to.NamespaceSelector == nil:
				sameNamespace = true
			}
		}
	}
	if !sameNamespace || !dns || !internet {
		t.Errorf("rules: own servers=%v dns=%v internet without private ranges=%v", sameNamespace, dns, internet)
	}
}
