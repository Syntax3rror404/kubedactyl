package clusterinfo

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/internal/tenancy"
)

// permissionCheck is one access the panel needs. Scope "" is cluster wide, "panel" the panel
// namespace and "tenant" a user namespace (where the panel's tenant role is bound).
type permissionCheck struct {
	label, verb, group, resource, sub, name, scope string
}

// errUnsure: a permission check failed or could not run yet; the result is shown but not cached.
var errUnsure = errors.New("permission checks incomplete")

// permissions verifies the access the panel needs (cached for permissionTTL).
func (s *Service) permissions(ctx context.Context) []Permission {
	p, _ := s.caches().permissionCache.Get(struct{}{}, func() ([]Permission, error) { return s.checkPermissions(ctx) })
	return p
}

// checkPermissions asks the API server for every access the panel needs (SelfSubjectAccessReview).
func (s *Service) checkPermissions(ctx context.Context) ([]Permission, error) {
	checks := s.permissionChecks()
	// The tenant role is bound per user namespace: check it in one of them (none yet: skipped).
	tenantNS := s.userNamespace(ctx)
	var unsure atomic.Bool
	if tenantNS == "" {
		checks = slices.DeleteFunc(checks, func(c permissionCheck) bool { return c.scope == "tenant" })
		unsure.Store(true)
	}
	out := make([]Permission, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		ns := map[string]string{"panel": s.Namespace, "tenant": tenantNS}[c.scope]
		resource := c.resource
		if c.sub != "" {
			resource += "/" + c.sub
		}
		out[i] = Permission{Label: c.label, Verb: c.verb, Group: c.group, Resource: resource, Namespace: ns}
		wg.Go(func() {
			attrs := &authzv1.ResourceAttributes{
				Verb: c.verb, Group: c.group, Resource: c.resource, Subresource: c.sub, Name: c.name, Namespace: ns,
			}
			spec := authzv1.SelfSubjectAccessReviewSpec{ResourceAttributes: attrs}
			res, err := s.Kube.Clientset.AuthorizationV1().SelfSubjectAccessReviews().
				Create(ctx, &authzv1.SelfSubjectAccessReview{Spec: spec}, metav1.CreateOptions{})
			out[i].Allowed = err == nil && res.Status.Allowed
			if err != nil {
				unsure.Store(true)
			}
		})
	}
	wg.Wait()
	if unsure.Load() {
		return out, errUnsure
	}
	return out, nil
}

// permissionChecks lists the access the panel needs with its current configuration.
func (s *Service) permissionChecks() []permissionCheck {
	const rbac = "rbac.authorization.k8s.io"
	checks := []permissionCheck{
		{label: "Install CRDs", verb: "create", group: "apiextensions.k8s.io", resource: "customresourcedefinitions"},
		{label: "Create namespaces", verb: "create", resource: "namespaces"},
		{label: "Delete namespaces", verb: "delete", resource: "namespaces"},
		{label: "Patch volumes (reclaim policy)", verb: "patch", resource: "persistentvolumes"},
		{label: "List nodes", verb: "list", resource: "nodes"},
		{label: "Read node metrics", verb: "list", group: "metrics.k8s.io", resource: "nodes"},
		{label: "List storage classes", verb: "list", group: "storage.k8s.io", resource: "storageclasses"},
		{label: "List load balancer pools", verb: "list", group: "cilium.io", resource: "ciliumloadbalancerippools"},
		{label: "Manage secrets", verb: "create", resource: "secrets", scope: "panel"},
		{label: "Start hardware probes", verb: "create", resource: "pods", scope: "panel"},
		{label: "Store hardware probe results", verb: "update", resource: "configmaps", name: hardwareCacheCM,
			scope: "panel"},
		{label: "Create pods", verb: "create", resource: "pods", scope: "tenant"},
		{label: "Exec into pods", verb: "create", resource: "pods", sub: "exec", scope: "tenant"},
		{label: "Attach to pods", verb: "create", resource: "pods", sub: "attach", scope: "tenant"},
		{label: "Read pod logs", verb: "get", resource: "pods", sub: "log", scope: "tenant"},
		{label: "Create services", verb: "create", resource: "services", scope: "tenant"},
		{label: "Create volume claims", verb: "create", resource: "persistentvolumeclaims", scope: "tenant"},
		{label: "Manage network policies", verb: "create", group: "networking.k8s.io", resource: "networkpolicies",
			scope: "tenant"},
	}
	if tenancy.TenantRole != "" {
		checks = append(checks,
			permissionCheck{label: "Grant server permissions", verb: "create", group: rbac, resource: "rolebindings"},
			permissionCheck{label: "Bind the tenant role", verb: "bind", group: rbac, resource: "clusterroles",
				name: tenancy.TenantRole},
		)
	}
	if s.SelfUpgrade {
		checks = append(checks, permissionCheck{
			label: "Start upgrade jobs", verb: "create", group: "batch", resource: "jobs", scope: "panel",
		})
	}
	return checks
}

// userNamespace returns a user namespace of this installation ("" when there is none yet).
func (s *Service) userNamespace(ctx context.Context) string {
	list, err := s.Kube.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: tenancy.LabelUser})
	if err != nil {
		return ""
	}
	for _, ns := range list.Items {
		if strings.HasPrefix(ns.Name, tenancy.NamespacePrefix) {
			return ns.Name
		}
	}
	return ""
}
