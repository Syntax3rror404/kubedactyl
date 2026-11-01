package clusterinfo

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/internal/tenancy"
)

// Hardware is what a probe pod reads from /proc and /sys/class/dmi on a node.
type Hardware struct {
	CPUModel       string `json:"cpuModel"                 example:"AMD Ryzen 7 5700U with Radeon Graphics"`
	CPUCores       int    `json:"cpuCores"                 example:"8"`
	CPUThreads     int    `json:"cpuThreads"               example:"16"`
	CPUSockets     int    `json:"cpuSockets"               example:"1"`
	CPUMaxMHz      int    `json:"cpuMaxMHz"                example:"4374"`
	Vendor         string `json:"vendor,omitempty"         example:"GMKtec"`
	Product        string `json:"product,omitempty"        example:"M5 Pro"`
	ProductVersion string `json:"productVersion,omitempty" example:"Version 1.0"`
	Board          string `json:"board,omitempty"`
	BIOS           string `json:"bios,omitempty"           example:"M5 Pro 1.03"`
	MemoryBytes    int64  `json:"memoryBytes"              example:"66750640128"`
	// Virtualized is true when the CPU reports the "hypervisor" flag (VM instead of bare metal).
	Virtualized bool `json:"virtualized"`
}

// probeScript prints key=value lines; it only reads world-readable files.
const probeScript = `c=/proc/cpuinfo
v() { grep -m1 "^$1" $c | cut -d: -f2- | sed 's/^ *//'; }
echo "cpu_model=$(v 'model name')"
echo "cpu_model_fallback=$(v 'Model')"
echo "cpu_threads=$(grep -c '^processor' $c)"
echo "cpu_cores=$(v 'cpu cores')"
echo "cpu_sockets=$(grep '^physical id' $c | sort -u | wc -l)"
echo "cpu_max_khz=$(cat /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq 2>/dev/null)"
for f in sys_vendor product_name product_version board_name bios_version; do
  echo "dmi_$f=$(cat /sys/class/dmi/id/$f 2>/dev/null)"
done
echo "mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')"
grep -qm1 hypervisor $c && echo virtualized=1 || echo virtualized=0
`

// parseProbe turns the probe output into Hardware.
func parseProbe(out string) *Hardware {
	kv := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok {
			kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	num := func(k string) int { n, _ := strconv.Atoi(kv[k]); return n }
	clean := func(s string) string {
		// Placeholder strings some BIOSes put into DMI fields.
		switch strings.ToLower(s) {
		case "", "default string", "to be filled by o.e.m.", "system product name", "not applicable", "none":
			return ""
		}
		return s
	}
	h := &Hardware{
		CPUModel:       cmp.Or(kv["cpu_model"], kv["cpu_model_fallback"]),
		CPUThreads:     num("cpu_threads"),
		CPUCores:       num("cpu_cores"),
		CPUSockets:     num("cpu_sockets"),
		CPUMaxMHz:      num("cpu_max_khz") / 1000,
		Vendor:         clean(kv["dmi_sys_vendor"]),
		Product:        clean(kv["dmi_product_name"]),
		ProductVersion: clean(kv["dmi_product_version"]),
		Board:          clean(kv["dmi_board_name"]),
		BIOS:           clean(kv["dmi_bios_version"]),
		Virtualized:    kv["virtualized"] == "1",
	}
	if kb, err := strconv.ParseInt(kv["mem_kb"], 10, 64); err == nil {
		h.MemoryBytes = kb * 1024
	}
	if h.CPUSockets == 0 && h.CPUThreads > 0 {
		h.CPUSockets = 1
	}
	if h.CPUCores > 0 {
		h.CPUCores *= h.CPUSockets // "cpu cores" is per socket
	} else {
		h.CPUCores = h.CPUThreads
	}
	return h
}

const (
	probeLabel       = "kubedactyl.io/role"
	probeRole        = "hardware-probe"
	hardwareCacheCM  = "kubedactyl-node-hardware"
	probeTimeout     = 90 * time.Second
	hardwareCacheTTL = 24 * time.Hour
)

// probeNode runs a short-lived, unprivileged pod pinned to the node and parses its output.
func (s *Service) probeNode(ctx context.Context, node string) (*Hardware, error) {
	pods := s.Kube.Clientset.CoreV1().Pods(s.Namespace)
	pod, err := pods.Create(ctx, s.probePod(node), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = pods.Delete(context.Background(), pod.Name, metav1.DeleteOptions{GracePeriodSeconds: ptr.To[int64](0)})
	}()
	return waitForProbe(ctx, pods, pod.Name)
}

// probePod is the unprivileged pod that runs probeScript on the node.
func (s *Service) probePod(node string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "kubedactyl-probe-", Labels: map[string]string{
			tenancy.LabelManagedBy: tenancy.ManagedBy, probeLabel: probeRole,
		}},
		Spec: corev1.PodSpec{
			NodeName:                     node,
			RestartPolicy:                corev1.RestartPolicyNever,
			ActiveDeadlineSeconds:        ptr.To[int64](60),
			AutomountServiceAccountToken: ptr.To(false),
			EnableServiceLinks:           ptr.To(false),
			Tolerations:                  []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: ptr.To[int64](65534), RunAsGroup: ptr.To[int64](65534), RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{{
				Name:    "probe",
				Image:   s.HelperImage,
				Command: []string{"sh", "-c", probeScript},
				Resources: corev1.ResourceRequirements{
					// Memory request = limit, CPU request = half the limit (same rule as server pods).
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("50m"),
						corev1.ResourceMemory: resource.MustParse("32Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("32Mi"),
					},
				},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(true),
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				},
			}},
		},
	}
}

// waitForProbe polls the probe pod until it finished and parses its output.
func waitForProbe(ctx context.Context, pods typedcorev1.PodInterface, name string) (*Hardware, error) {
	deadline := time.Now().Add(probeTimeout)
	for time.Now().Before(deadline) {
		p, err := pods.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		switch p.Status.Phase {
		case corev1.PodSucceeded:
			raw, err := pods.GetLogs(name, &corev1.PodLogOptions{Container: "probe"}).DoRaw(ctx)
			if err != nil {
				return nil, err
			}
			return parseProbe(string(raw)), nil
		case corev1.PodFailed:
			return nil, fmt.Errorf("probe pod failed: %s", cmp.Or(p.Status.Message, p.Status.Reason))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, errors.New("probe timed out (is the node reachable?)")
}

// loadCache reads the persisted probe results once, so a panel restart does not probe again, and removes the
// probe pods an earlier panel left behind. It reports whether the results are loaded; after a failed read it tries
// again on the next call, and nothing is probed until then (a probe would overwrite the stored results).
func (s *Service) loadCache(ctx context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return true
	}
	cm, err := s.Kube.Clientset.CoreV1().ConfigMaps(s.Namespace).Get(ctx, hardwareCacheCM, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		s.Log.Warn("reading hardware cache", "err", err)
		return false
	}
	s.hardware = map[string]*hardwareEntry{}
	if err == nil {
		for node, raw := range cm.Data {
			e := &hardwareEntry{}
			if json.Unmarshal([]byte(raw), e) == nil {
				s.hardware[node] = e
			}
		}
	}
	// Before the first probe of this panel, so every probe pod found here is a leftover.
	s.removeProbePods(ctx)
	s.loaded = true
	return true
}

// removeProbePods deletes the probe pods in the panel namespace.
func (s *Service) removeProbePods(ctx context.Context) {
	pods := s.Kube.Clientset.CoreV1().Pods(s.Namespace)
	list, err := pods.List(ctx, metav1.ListOptions{LabelSelector: probeLabel + "=" + probeRole})
	if err != nil {
		s.Log.Warn("listing old hardware probe pods", "err", err)
		return
	}
	for _, p := range list.Items {
		err := pods.Delete(ctx, p.Name, metav1.DeleteOptions{GracePeriodSeconds: ptr.To[int64](0)})
		if client.IgnoreNotFound(err) != nil {
			s.Log.Warn("removing an old hardware probe pod", "pod", p.Name, "err", err)
		}
	}
}

// forgetGoneNodes drops the results of nodes that left the cluster, so they are not saved again.
func (s *Service) forgetGoneNodes(nodes []corev1.Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name := range s.hardware {
		if !slices.ContainsFunc(nodes, func(n corev1.Node) bool { return n.Name == name }) {
			delete(s.hardware, name)
		}
	}
}

// saveCache persists the current probe results.
func (s *Service) saveCache(ctx context.Context) {
	s.mu.Lock()
	data := map[string]string{}
	for node, e := range s.hardware {
		if raw, err := json.Marshal(e); err == nil {
			data[node] = string(raw)
		}
	}
	s.mu.Unlock()
	cms := s.Kube.Clientset.CoreV1().ConfigMaps(s.Namespace)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   hardwareCacheCM,
			Labels: map[string]string{tenancy.LabelManagedBy: tenancy.ManagedBy},
		},
		Data: data,
	}
	_, err := cms.Update(ctx, cm, metav1.UpdateOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cms.Create(ctx, cm, metav1.CreateOptions{})
	}
	if err != nil {
		s.Log.Warn("saving hardware cache", "err", err)
	}
}

// ProbeNodes probes the given nodes in parallel and stores the results.
func (s *Service) ProbeNodes(ctx context.Context, nodes []string) {
	if !s.loadCache(ctx) {
		return
	}
	var wg sync.WaitGroup
	for _, node := range nodes {
		s.mu.Lock()
		e := s.hardware[node]
		if e != nil && e.probing {
			s.mu.Unlock()
			continue
		}
		if e == nil {
			e = &hardwareEntry{}
			s.hardware[node] = e
		}
		e.probing = true
		s.mu.Unlock()

		wg.Go(func() {
			hw, err := s.probeNode(ctx, node)
			s.mu.Lock()
			defer s.mu.Unlock()
			e.probing = false
			e.ProbedAt = time.Now().UTC()
			if err != nil {
				e.Error = err.Error()
				s.Log.Warn("hardware probe failed", "node", node, "err", err)
				return
			}
			e.Hardware, e.Error = hw, ""
		})
	}
	wg.Wait()
	s.saveCache(context.WithoutCancel(ctx))
}

// hardwareFor returns a copy of the cached result of a node and whether it needs a new probe.
func (s *Service) hardwareFor(node string) (e hardwareEntry, stale bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cached := s.hardware[node]; cached != nil {
		e = *cached
	}
	return e, e.ProbedAt.IsZero() || time.Since(e.ProbedAt) > hardwareCacheTTL
}
