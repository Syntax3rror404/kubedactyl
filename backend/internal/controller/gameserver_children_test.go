package controller

import (
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestLoadBalancerIPsFollowFamilies(t *testing.T) {
	svc := &corev1.Service{}
	svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
		{IP: "2001:db8::5"}, {Hostname: "lb"}, {IP: "192.0.2.5"},
	}
	svc.Spec.IPFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
	if got := loadBalancerIPs(svc); !slices.Equal(got, []string{"192.0.2.5", "2001:db8::5"}) {
		t.Errorf("IPv4 first: %v", got)
	}
	svc.Spec.IPFamilies = []corev1.IPFamily{corev1.IPv6Protocol, corev1.IPv4Protocol}
	if got := loadBalancerIPs(svc); !slices.Equal(got, []string{"2001:db8::5", "192.0.2.5"}) {
		t.Errorf("IPv6 first: %v", got)
	}
}
