package gameserver

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/netip"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"app/api/v1alpha1"
	"app/internal/tenancy"
)

func labels(server, role string) map[string]string {
	return map[string]string{
		LabelServer:              server,
		LabelRole:                role,
		"app.kubernetes.io/name": "kubedactyl",
		tenancy.LabelManagedBy:   tenancy.ManagedBy,
	}
}

func mib(v int64) resource.Quantity { return *resource.NewQuantity(v<<20, resource.BinarySI) }

// memoryLimitMiB adds an overhead for the runtime to the container memory limit, so the game
// can use the memory it was given (SERVER_MEMORY).
func memoryLimitMiB(memory int64) int64 {
	multiplier := 1.05
	switch {
	case memory <= 2048:
		multiplier = 1.15
	case memory <= 4096:
		multiplier = 1.10
	}
	return int64(float64(memory) * multiplier)
}

// PVC is the persistent server data volume.
func PVC(gs *v1alpha1.GameServer, opts Options) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: PVCName(gs.Name), Namespace: gs.Namespace, Labels: labels(gs.Name, "data")},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: ptr.To(cmp.Or(gs.Spec.StorageClass, opts.StorageClass)),
			VolumeName:       gs.Annotations[AnnotationMovedVolume],
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: mib(gs.Spec.Resources.DiskMiB)},
			},
		},
	}
}

// Service exposes every allocation as TCP and UDP. poolLabels are the
// labels the service needs to get its address from the selected load balancer pool.
// A ClusterIP server gets neither pool labels nor fixed IPs.
func Service(gs *v1alpha1.GameServer, poolLabels map[string]string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ServiceName(gs.Name),
			Namespace:   gs.Namespace,
			Labels:      labels(gs.Name, "network"),
			Annotations: map[string]string{},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			Selector:       map[string]string{LabelServer: gs.Name, LabelRole: RoleGame},
			IPFamilyPolicy: ipFamilyPolicy(gs.Spec.IPv6),
		},
	}
	if !ClusterOnly(gs) {
		svc.Spec.Type = corev1.ServiceTypeLoadBalancer
		// Local by default: the game server sees the players' IPs (ban lists etc.).
		svc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicy(
			cmp.Or(gs.Spec.ExternalTrafficPolicy, v1alpha1.TrafficLocal),
		)
		if gs.Spec.LoadBalancerIP != "" {
			svc.Annotations[AnnotationFixedIPs] = FixedIPs(gs.Spec.LoadBalancerIP, nil)
		}
		maps.Copy(svc.Labels, poolLabels)
		if len(poolLabels) > 0 {
			svc.Annotations[AnnotationPoolLabels] = strings.Join(slices.Sorted(maps.Keys(poolLabels)), ",")
		}
	}
	for _, p := range gs.Spec.Ports {
		for _, proto := range []corev1.Protocol{corev1.ProtocolTCP, corev1.ProtocolUDP} {
			svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
				Name:       fmt.Sprintf("%s-%d", strings.ToLower(string(proto)), p),
				Protocol:   proto,
				Port:       p,
				TargetPort: intstr.FromInt32(p),
			})
		}
	}
	return svc
}

// ClusterOnly tells whether a server is published only inside the cluster (no load balancer).
func ClusterOnly(gs *v1alpha1.GameServer) bool {
	return gs.Spec.ServiceType == v1alpha1.ServiceClusterIP
}

// ClusterAddress is the DNS name of a server's service inside the cluster.
func ClusterAddress(gs *v1alpha1.GameServer) string {
	return ServiceName(gs.Name) + "." + gs.Namespace + ".svc"
}

// ParseIPs parses a comma separated list of IPs (the fixed IPs of a server).
func ParseIPs(list string) ([]netip.Addr, error) {
	var addrs []netip.Addr
	for ip := range strings.SplitSeq(list, ",") {
		addr, err := netip.ParseAddr(strings.TrimSpace(ip))
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

// IPFamily is the family of an address.
func IPFamily(addr netip.Addr) corev1.IPFamily {
	if addr.Is6() {
		return corev1.IPv6Protocol
	}
	return corev1.IPv4Protocol
}

// FixedIPs keeps the fixed IPs (comma separated) of the service's IP families. Cilium assigns
// requested IPs of any family, so an IPv6 address would stay on a service without IPv6. A new
// service has no families yet: then all are kept, the next reconcile drops the others. An invalid
// list (the API rejects it) requests none.
func FixedIPs(fixed string, families []corev1.IPFamily) string {
	addrs, _ := ParseIPs(fixed)
	var keep []string
	for _, addr := range addrs {
		if len(families) == 0 || slices.Contains(families, IPFamily(addr)) {
			keep = append(keep, addr.String())
		}
	}
	return strings.Join(keep, ",")
}

// LoadBalancerIPs are the IPs the load balancer assigned to the service (none while pending) in
// the order of its IP families, so the first is of the cluster's main family (Cilium lists fixed
// IPs in the order they were given).
func LoadBalancerIPs(svc *corev1.Service) []string {
	var addrs []netip.Addr
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if addr, err := netip.ParseAddr(ing.IP); err == nil {
			addrs = append(addrs, addr)
		}
	}
	rank := func(addr netip.Addr) int {
		if i := slices.Index(svc.Spec.IPFamilies, IPFamily(addr)); i >= 0 {
			return i
		}
		return len(svc.Spec.IPFamilies)
	}
	slices.SortStableFunc(addrs, func(a, b netip.Addr) int { return cmp.Compare(rank(a), rank(b)) })
	ips := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.String())
	}
	return ips
}

// ipFamilyPolicy is PreferDualStack with IPv6: the API server keeps only the families the
// cluster has, so single stack clusters are unaffected. Switching back to SingleStack makes the
// API server release the second family.
func ipFamilyPolicy(ipv6 bool) *corev1.IPFamilyPolicy {
	if ipv6 {
		return ptr.To(corev1.IPFamilyPolicyPreferDualStack)
	}
	return ptr.To(corev1.IPFamilyPolicySingleStack)
}

// sameNodeAsVolume keeps all pods that mount the data volume on one node, because the
// RWO volume can only be attached to one node at a time. Every such pod carries
// LabelVolume and matches its own affinity term, so the first one may go to any node
// (the scheduler allows that when no other pod matches) and later ones join it.
// Terminated pods are not considered. Everything else is left to the Kubernetes
// scheduler, based on the resource requests below.
func sameNodeAsVolume(server string) *corev1.Affinity {
	return &corev1.Affinity{PodAffinity: &corev1.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelVolume: server}},
			TopologyKey:   "kubernetes.io/hostname",
		}},
	}}
}

// resources reserves memory in full (request = limit, memory cannot be taken back) and
// half of the CPU limit. With real requests the scheduler sees how full each node is and
// spreads the pods; short CPU bursts up to the limit stay possible.
func resources(cpuLimitMillis, memoryMiB int64) corev1.ResourceRequirements {
	cpuLimit := resource.NewMilliQuantity(cpuLimitMillis, resource.DecimalSI)
	cpuRequest := resource.NewMilliQuantity(max(cpuLimitMillis/2, 10), resource.DecimalSI)
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: *cpuRequest, corev1.ResourceMemory: mib(memoryMiB)},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: *cpuLimit, corev1.ResourceMemory: mib(memoryMiB)},
	}
}

// UnlimitedCPURequestMillis is reserved for servers without CPU limit (cpuMillis 0),
// so the scheduler still accounts for them.
const UnlimitedCPURequestMillis = 1000

// volumeLabels are the labels of a pod that mounts the data volume.
func volumeLabels(server, role string) map[string]string {
	l := labels(server, role)
	l[LabelVolume] = server
	return l
}

func dataVolume(server string) corev1.Volume {
	return corev1.Volume{Name: "data", VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: PVCName(server)},
	}}
}

// tmpVolume is a 100 MiB tmpfs at /tmp (the root file system is read-only).
func tmpVolume() corev1.Volume {
	return corev1.Volume{Name: "tmp", VolumeSource: corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory, SizeLimit: ptr.To(mib(100))},
	}}
}

// RuntimeHash fingerprints the settings that only apply when the game pod is created: image,
// the environment (effective startup command, variables with the egg defaults, memory, port),
// CPU, ports and the egg's config files (written before every start). A running pod with
// another hash needs a restart to pick up the changes, also after the egg was edited.
func RuntimeHash(gs *v1alpha1.GameServer, e *v1alpha1.Egg, opts Options) string {
	s := gs.Spec
	data, _ := json.Marshal(
		[]any{s.Image, Environment(gs, e, opts), s.Resources.CPUMillis, s.Ports, e.Spec.ConfigFiles},
	)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}
