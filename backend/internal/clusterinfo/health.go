package clusterinfo

import (
	"context"
	"slices"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
	"app/internal/checks"
	"app/internal/settings"
	"app/internal/sso"
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
	oidcErr        error // why the identity provider of single sign-on does not answer
	// guarded is true when the panel runs with its tenant role (in the cluster). policy limits its
	// cluster wide permissions, policyErr says why it cannot be read.
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

// ForgetHealth makes the next Health check again, after a change the checks depend on (the settings).
func (s *Service) ForgetHealth() {
	s.caches().healthCache.Forget(struct{}{})
}

// gatherHealth reads everything the checks look at. The reads do not depend on each other, so they run
// in parallel and each fills in its own fields:
//  1. API groups: the own CRDs, metrics-server and Cilium (then its pools).
//  2. Permissions and nodes.
//  3. Admission policy (only in the cluster).
//  4. Settings with the identity provider they name, and the storage classes.
func (s *Service) gatherHealth(ctx context.Context) healthInputs {
	in := healthInputs{guarded: tenancy.TenantRole != "", policy: s.AdmissionPolicy}
	var wg sync.WaitGroup
	// 1. API groups
	wg.Go(func() { s.readAPIGroups(ctx, &in) })
	// 2. Permissions and nodes
	wg.Go(func() { in.permissions = s.permissions(ctx) })
	wg.Go(func() {
		list, err := s.Kube.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if in.nodesErr = err; err == nil {
			in.nodes = list.Items
		}
	})
	// 3. Admission policy
	if policy := in.policy; in.guarded && policy != "" {
		wg.Go(func() { in.policyErr = s.readAdmissionPolicy(ctx, policy) })
	}
	// 4. Settings
	if s.Settings != nil {
		wg.Go(func() {
			if in.settings, in.settingsErr = s.Settings.Current(ctx); in.settings.OIDC.Enabled {
				in.oidcErr = sso.Discover(ctx, in.settings.OIDC.IssuerURL)
			}
		})
		wg.Go(func() { in.storageClasses, in.storageErr = settings.ListStorageClasses(ctx, s.Reader) })
	}
	wg.Wait()
	return in
}

// readAPIGroups finds the resources of the own CRDs, metrics-server and Cilium, then reads Cilium's pools.
func (s *Service) readAPIGroups(ctx context.Context, in *healthInputs) {
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
	in.cilium = slices.ContainsFunc([]string{"cilium.io/v2", "cilium.io/v2alpha1"}, func(gv string) bool {
		list, err := disc.ServerResourcesForGroupVersion(gv)
		return err == nil && slices.ContainsFunc(list.APIResources, func(r metav1.APIResource) bool {
			return r.Name == "ciliumloadbalancerippools"
		})
	})
	if in.cilium && s.Settings != nil {
		in.pools, in.poolsErr = settings.ListPoolsWithUsage(ctx, s.Reader)
	}
}

// readAdmissionPolicy checks that the admission policy and its binding exist.
func (s *Service) readAdmissionPolicy(ctx context.Context, name string) error {
	adm := s.Kube.Clientset.AdmissionregistrationV1()
	if _, err := adm.ValidatingAdmissionPolicies().Get(ctx, name, metav1.GetOptions{}); err != nil {
		return err
	}
	_, err := adm.ValidatingAdmissionPolicyBindings().Get(ctx, name, metav1.GetOptions{})
	return err
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
		checkOIDC(in),
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
