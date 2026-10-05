package clusterinfo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Usage is live resource usage from metrics-server.
type Usage struct {
	CPUMillis   int64 `json:"cpuMillis"   example:"1250"`
	MemoryBytes int64 `json:"memoryBytes" example:"12884901888"`
}

// Node describes one cluster node.
type Node struct {
	Name          string    `json:"name"          example:"node-1"`
	Roles         []string  `json:"roles"         example:"control-plane"`
	Ready         bool      `json:"ready"`
	Status        string    `json:"status"        example:"Ready"`
	Unschedulable bool      `json:"unschedulable"`
	Pressure      []string  `json:"pressure"      example:"MemoryPressure"`
	InternalIP    string    `json:"internalIP"    example:"192.168.1.41"`
	CreatedAt     time.Time `json:"createdAt"`
	Taints        []string  `json:"taints"`

	Architecture     string `json:"architecture"     example:"amd64"`
	OSImage          string `json:"osImage"          example:"Talos (v1.14.1)"`
	KernelVersion    string `json:"kernelVersion"    example:"6.18.51-talos"`
	KubeletVersion   string `json:"kubeletVersion"   example:"v1.36.4"`
	ContainerRuntime string `json:"containerRuntime" example:"containerd://2.3.5"`

	CPUCapacityMillis      int64 `json:"cpuCapacityMillis"      example:"16000"`
	CPUAllocatableMillis   int64 `json:"cpuAllocatableMillis"`
	CPURequestedMillis     int64 `json:"cpuRequestedMillis"`
	MemoryCapacityBytes    int64 `json:"memoryCapacityBytes"`
	MemoryAllocatableBytes int64 `json:"memoryAllocatableBytes"`
	MemoryRequestedBytes   int64 `json:"memoryRequestedBytes"`
	EphemeralStorageBytes  int64 `json:"ephemeralStorageBytes"`
	PodsCapacity           int64 `json:"podsCapacity"           example:"110"`
	PodsRunning            int   `json:"podsRunning"            example:"42"`

	Usage *Usage `json:"usage,omitempty"`

	Hardware         *Hardware  `json:"hardware,omitempty"`
	HardwareProbedAt *time.Time `json:"hardwareProbedAt,omitempty"`
	HardwareError    string     `json:"hardwareError,omitempty"`
	HardwareProbing  bool       `json:"hardwareProbing"`
}

// CPUModelCount groups identical CPUs.
type CPUModelCount struct {
	Model string `json:"model" example:"AMD Ryzen 7 5700U with Radeon Graphics"`
	Count int    `json:"count" example:"6"`
}

// Totals sums up all nodes.
type Totals struct {
	Nodes                int             `json:"nodes"`
	ReadyNodes           int             `json:"readyNodes"`
	CPUCapacityMillis    int64           `json:"cpuCapacityMillis"`
	CPURequestedMillis   int64           `json:"cpuRequestedMillis"`
	CPUUsedMillis        int64           `json:"cpuUsedMillis"`
	PhysicalCores        int             `json:"physicalCores"`
	MemoryCapacityBytes  int64           `json:"memoryCapacityBytes"`
	MemoryRequestedBytes int64           `json:"memoryRequestedBytes"`
	MemoryUsedBytes      int64           `json:"memoryUsedBytes"`
	PodsRunning          int             `json:"podsRunning"`
	PodsCapacity         int64           `json:"podsCapacity"`
	CPUModels            []CPUModelCount `json:"cpuModels"`
	// MetricsAvailable is false when metrics-server does not answer.
	MetricsAvailable bool `json:"metricsAvailable"`
}

// NodesResponse is the node overview.
type NodesResponse struct {
	Nodes  []Node `json:"nodes"`
	Totals Totals `json:"totals"`
}

type nodeMetricsList struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Usage map[string]string `json:"usage"`
	} `json:"items"`
}

// Nodes returns all nodes with capacity, requests, usage and cached hardware. Ready nodes
// without (fresh) hardware data are probed in the background.
func (s *Service) Nodes(ctx context.Context) (*NodesResponse, error) {
	s.loadCache(ctx)
	data, err := s.caches().nodeCache.Get(struct{}{}, func() (*nodeData, error) { return s.readNodes(ctx) })
	if err != nil {
		return nil, err
	}

	resp := &NodesResponse{Nodes: []Node{}}
	var toProbe []string
	for _, n := range data.nodes {
		node := buildNode(&n)
		if a := data.perNode[n.Name]; a != nil {
			node.CPURequestedMillis, node.MemoryRequestedBytes, node.PodsRunning = a.cpu, a.mem, a.pods
		}
		node.Usage = data.usage[n.Name]
		hw, probedAt, errMsg, probing, stale := s.hardwareFor(n.Name)
		node.Hardware, node.HardwareProbedAt, node.HardwareError, node.HardwareProbing = hw, probedAt, errMsg, probing
		if stale && !probing && node.Ready && !n.Spec.Unschedulable {
			toProbe = append(toProbe, n.Name)
			node.HardwareProbing = true
		}
		resp.Nodes = append(resp.Nodes, node)
	}
	sort.Slice(resp.Nodes, func(i, j int) bool { return resp.Nodes[i].Name < resp.Nodes[j].Name })
	resp.Totals = totals(resp.Nodes, data.metricsOK)
	if len(toProbe) > 0 {
		go s.ProbeNodes(context.WithoutCancel(ctx), toProbe)
	}
	return resp, nil
}

// nodeData is what Nodes reads from the API server (cached for nodeTTL); the hardware is added per request.
type nodeData struct {
	nodes     []corev1.Node
	perNode   map[string]*allocation
	usage     map[string]*Usage
	metricsOK bool
}

func (s *Service) readNodes(ctx context.Context) (*nodeData, error) {
	list, err := s.Kube.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	perNode, err := s.allocations(ctx)
	if err != nil {
		return nil, err
	}
	usage, metricsOK := s.usage(ctx)
	return &nodeData{nodes: list.Items, perNode: perNode, usage: usage, metricsOK: metricsOK}, nil
}

// allocation sums the requests of the running pods on a node.
type allocation struct {
	cpu, mem int64
	pods     int
}

// allocations returns the requests of the running pods per node.
func (s *Service) allocations(ctx context.Context) (map[string]*allocation, error) {
	pods, err := s.Kube.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "status.phase=Running"})
	if err != nil {
		return nil, err
	}
	perNode := map[string]*allocation{}
	for _, p := range pods.Items {
		a := perNode[p.Spec.NodeName]
		if a == nil {
			a = &allocation{}
			perNode[p.Spec.NodeName] = a
		}
		a.pods++
		for _, c := range p.Spec.Containers {
			a.cpu += c.Resources.Requests.Cpu().MilliValue()
			a.mem += c.Resources.Requests.Memory().Value()
		}
	}
	return perNode, nil
}

// usage returns the live usage per node from metrics-server; ok is false when it is not available.
func (s *Service) usage(ctx context.Context) (usage map[string]*Usage, ok bool) {
	usage = map[string]*Usage{}
	raw, err := s.Kube.Clientset.CoreV1().RESTClient().Get().AbsPath("/apis/metrics.k8s.io/v1beta1/nodes").DoRaw(ctx)
	var ml nodeMetricsList
	ok = err == nil && json.Unmarshal(raw, &ml) == nil
	for _, m := range ml.Items {
		u := &Usage{}
		if q, err := resource.ParseQuantity(m.Usage["cpu"]); err == nil {
			u.CPUMillis = q.MilliValue()
		}
		if q, err := resource.ParseQuantity(m.Usage["memory"]); err == nil {
			u.MemoryBytes = q.Value()
		}
		usage[m.Metadata.Name] = u
	}
	return usage, ok
}

func buildNode(n *corev1.Node) Node {
	node := Node{
		Name: n.Name, CreatedAt: n.CreationTimestamp.Time, Unschedulable: n.Spec.Unschedulable,
		Roles: []string{}, Pressure: []string{}, Taints: []string{}, Status: "Unknown",
		Architecture: n.Status.NodeInfo.Architecture, OSImage: n.Status.NodeInfo.OSImage,
		KernelVersion: n.Status.NodeInfo.KernelVersion, KubeletVersion: n.Status.NodeInfo.KubeletVersion,
		ContainerRuntime:       n.Status.NodeInfo.ContainerRuntimeVersion,
		CPUCapacityMillis:      n.Status.Capacity.Cpu().MilliValue(),
		CPUAllocatableMillis:   n.Status.Allocatable.Cpu().MilliValue(),
		MemoryCapacityBytes:    n.Status.Capacity.Memory().Value(),
		MemoryAllocatableBytes: n.Status.Allocatable.Memory().Value(),
		EphemeralStorageBytes:  n.Status.Capacity.StorageEphemeral().Value(),
		PodsCapacity:           n.Status.Capacity.Pods().Value(),
	}
	for label := range n.Labels {
		if role, ok := strings.CutPrefix(label, "node-role.kubernetes.io/"); ok && role != "" {
			node.Roles = append(node.Roles, role)
		}
	}
	sort.Strings(node.Roles)
	if len(node.Roles) == 0 {
		node.Roles = append(node.Roles, "worker")
	}
	for _, c := range n.Status.Conditions {
		switch {
		case c.Type == corev1.NodeReady:
			node.Ready = c.Status == corev1.ConditionTrue
			node.Status = map[corev1.ConditionStatus]string{
				corev1.ConditionTrue:  "Ready",
				corev1.ConditionFalse: "NotReady",
			}[c.Status]
			if node.Status == "" {
				node.Status = "Unknown"
			}
		case c.Status == corev1.ConditionTrue && strings.HasSuffix(string(c.Type), "Pressure"):
			node.Pressure = append(node.Pressure, string(c.Type))
		}
	}
	for _, a := range n.Status.Addresses {
		if a.Type == corev1.NodeInternalIP && node.InternalIP == "" {
			node.InternalIP = a.Address
		}
	}
	for _, t := range n.Spec.Taints {
		s := t.Key
		if t.Value != "" {
			s += "=" + t.Value
		}
		node.Taints = append(node.Taints, s+":"+string(t.Effect))
	}
	return node
}

func totals(nodes []Node, metricsOK bool) Totals {
	t := Totals{Nodes: len(nodes), MetricsAvailable: metricsOK, CPUModels: []CPUModelCount{}}
	models := map[string]int{}
	for _, n := range nodes {
		if n.Ready {
			t.ReadyNodes++
		}
		t.CPUCapacityMillis += n.CPUCapacityMillis
		t.CPURequestedMillis += n.CPURequestedMillis
		t.MemoryCapacityBytes += n.MemoryCapacityBytes
		t.MemoryRequestedBytes += n.MemoryRequestedBytes
		t.PodsRunning += n.PodsRunning
		t.PodsCapacity += n.PodsCapacity
		if n.Usage != nil {
			t.CPUUsedMillis += n.Usage.CPUMillis
			t.MemoryUsedBytes += n.Usage.MemoryBytes
		}
		if n.Hardware != nil {
			t.PhysicalCores += n.Hardware.CPUCores
			if n.Hardware.CPUModel != "" {
				models[n.Hardware.CPUModel]++
			}
		}
	}
	for m, c := range models {
		t.CPUModels = append(t.CPUModels, CPUModelCount{Model: m, Count: c})
	}
	sort.Slice(t.CPUModels, func(i, j int) bool { return t.CPUModels[i].Count > t.CPUModels[j].Count })
	return t
}
