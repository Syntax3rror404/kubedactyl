package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"app/api/v1alpha1"
	"app/internal/kube"
	"app/internal/tenancy"
)

// NetworkPolicyName is the egress policy in every user namespace.
const NetworkPolicyName = "kubedactyl-isolation"

// Private, shared and link-local ranges: the cluster, the LAN, node and cloud metadata addresses.
var privateV4 = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"100.64.0.0/10",
	"169.254.0.0/16",
	"127.0.0.0/8",
}
var privateV6 = []string{"fc00::/7", "fe80::/10", "::1/128"}

// IsolationPolicy lets the pods of a user namespace reach the internet, the cluster DNS
// and each other, but not other namespaces, nodes, the Kubernetes API or the LAN. Game
// servers run user supplied files (plugins, mods), so they are treated as untrusted.
// Incoming traffic (players via the load balancer) is not restricted.
func IsolationPolicy(namespace string) *networkingv1.NetworkPolicy {
	udp, tcp := corev1.ProtocolUDP, corev1.ProtocolTCP
	dns := intstr.FromInt32(53)
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NetworkPolicyName,
			Namespace: namespace,
			Labels:    map[string]string{tenancy.LabelManagedBy: tenancy.ManagedBy},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{To: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}}},
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"},
							},
							PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &udp, Port: &dns}, {Protocol: &tcp, Port: &dns}},
				},
				{To: []networkingv1.NetworkPolicyPeer{
					{IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0", Except: privateV4}},
					{IPBlock: &networkingv1.IPBlock{CIDR: "::/0", Except: privateV6}},
				}},
			},
		},
	}
}

// ensureNetworkPolicy applies or removes the isolation policy according to the panel settings.
func (r *UserReconciler) ensureNetworkPolicy(ctx context.Context, namespace string) error {
	settings := &v1alpha1.PanelSettings{}
	if err := r.Reader.Get(
		ctx, client.ObjectKey{Namespace: tenancy.SystemNamespace, Name: v1alpha1.SettingsName}, settings,
	); client.IgnoreNotFound(
		err,
	) != nil {
		return err
	}
	cur := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: NetworkPolicyName, Namespace: namespace}}
	if settings.Spec.AllowPrivateNetworks {
		return client.IgnoreNotFound(r.Delete(ctx, cur))
	}
	want := IsolationPolicy(namespace)
	err := kube.CreateOrPatch(ctx, r.Reader, r.Client, cur, func() error {
		cur.Labels = want.Labels
		cur.Spec = want.Spec
		return nil
	})
	if apierrors.IsNotFound(err) {
		return nil // namespace is being deleted
	}
	return err
}

// usersForSettings re-reconciles all users when the panel settings change.
func (r *UserReconciler) usersForSettings(ctx context.Context, _ client.Object) []reconcile.Request {
	var users v1alpha1.UserList
	if err := r.List(ctx, &users, client.InNamespace(tenancy.SystemNamespace)); err != nil {
		return nil
	}
	out := make([]reconcile.Request, 0, len(users.Items))
	for _, u := range users.Items {
		out = append(out, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&u)})
	}
	return out
}
