package clusterinfo

import (
	"context"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/checks"
	"app/internal/settings"
	"app/internal/tenancy"
)

// Health summarizes the checks; Status is the worst of them.
type Health struct {
	Status    checks.Status  `json:"status"    enums:"ok,warning,error"`
	Checks    []checks.Check `json:"checks"`
	CheckedAt time.Time      `json:"checkedAt"`
}

// healthInputs is everything the checks look at, gathered from the cluster.
type healthInputs struct {
	crdResources   []string // resources served by kubedactyl.io/v1alpha1
	crdErr         error
	permissions    []Permission
	nodes          []corev1.Node
	nodesErr       error
	metrics        bool
	cilium         bool
	settings       v1alpha1.PanelSettingsSpec
	settingsErr    error
	storageClasses []settings.StorageClass
	storageErr     error
	pools          []settings.Pool
	poolsErr       error
	// guarded: the panel runs with its tenant role (in the cluster); policy is the admission
	// policy that limits its cluster wide permissions, policyErr why it cannot be read.
	guarded   bool
	policy    string
	policyErr error
}

// Health checks the dependencies of the panel (cached for 30 seconds).
func (s *Service) Health(ctx context.Context) *Health {
	h, _ := s.caches().healthCache.Get(struct{}{}, func() (*Health, error) {
		h := &Health{Checks: evaluate(s.gatherHealth(ctx)), CheckedAt: time.Now()}
		h.Status = checks.Worst(h.Checks)
		return h, nil
	})
	return h
}

func (s *Service) gatherHealth(ctx context.Context) healthInputs {
	var in healthInputs
	disc := s.Kube.Clientset.Discovery()
	if list, err := disc.ServerResourcesForGroupVersion(v1alpha1.GroupVersion.String()); err != nil {
		in.crdErr = err
	} else {
		for _, r := range list.APIResources {
			in.crdResources = append(in.crdResources, r.Name)
		}
	}
	_, err := disc.ServerResourcesForGroupVersion("metrics.k8s.io/v1beta1")
	in.metrics = err == nil
	for _, gv := range []string{"cilium.io/v2", "cilium.io/v2alpha1"} {
		if list, err := disc.ServerResourcesForGroupVersion(gv); err == nil {
			in.cilium = in.cilium ||
				slices.ContainsFunc(
					list.APIResources,
					func(r metav1.APIResource) bool { return r.Name == "ciliumloadbalancerippools" },
				)
		}
	}
	in.permissions = s.permissions(ctx)
	if nodes, err := s.Kube.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err != nil {
		in.nodesErr = err
	} else {
		in.nodes = nodes.Items
	}
	in.guarded, in.policy = tenancy.TenantRole != "", s.AdmissionPolicy
	if in.guarded && in.policy != "" {
		adm := s.Kube.Clientset.AdmissionregistrationV1()
		if _, err := adm.ValidatingAdmissionPolicies().Get(ctx, in.policy, metav1.GetOptions{}); err != nil {
			in.policyErr = err
		} else if _, err := adm.ValidatingAdmissionPolicyBindings().
			Get(ctx, in.policy, metav1.GetOptions{}); err != nil {
			in.policyErr = err
		}
	}
	if s.Settings != nil {
		in.settings, in.settingsErr = s.Settings.Get(ctx)
		in.storageClasses, in.storageErr = settings.ListStorageClasses(ctx, s.Reader)
		if in.cilium {
			in.pools, in.poolsErr = settings.ListPools(ctx, s.Reader)
		}
	}
	return in
}

// evaluate turns the gathered facts into checks (no I/O, unit tested). New external
// dependency? Add a check here (and a function in health_checks.go).
func evaluate(in healthInputs) []checks.Check {
	return []checks.Check{
		checkCRDs(in),
		checkPermissions(in),
		checkNodes(in),
		checkMetrics(in),
		checkLoadBalancer(in),
		checkStorage(in),
		checkAdmissionPolicy(in),
	}
}

func errText(errs ...error) string {
	for _, err := range errs {
		if err != nil {
			return err.Error()
		}
	}
	return ""
}
