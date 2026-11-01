package clusterinfo

import (
	"errors"
	"testing"

	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"app/internal/kube"
	"app/internal/settings"
	"app/internal/tenancy"
	"app/internal/testutil"
)

func TestCheckPermissions(t *testing.T) {
	testutil.Builder(t) // sets the namespaces of the test installation
	userNS := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: tenancy.NamespacePrefix + "alice", Labels: map[string]string{tenancy.LabelUser: "alice"},
	}}
	cs := k8sfake.NewClientset(userNS)
	cs.PrependReactor("create", "selfsubjectaccessreviews",
		func(a k8stesting.Action) (bool, runtime.Object, error) {
			review := a.(k8stesting.CreateAction).GetObject().(*authzv1.SelfSubjectAccessReview)
			if review.Spec.ResourceAttributes.Subresource == "attach" {
				return true, nil, errors.New("timeout")
			}
			review.Status.Allowed = true
			return true, review, nil
		})
	s := &Service{Kube: &kube.Client{Clientset: cs}, Namespace: testutil.Namespace}
	got, err := s.checkPermissions(t.Context())
	if !errors.Is(err, errUnsure) {
		t.Errorf("a failed check must not be kept: %v", err)
	}
	byLabel := map[string]Permission{}
	for _, p := range got {
		byLabel[p.Label] = p
	}
	cases := []struct {
		label, namespace string
		allowed, unknown bool
	}{
		{"Read the Cilium config", settings.CiliumConfig.Namespace, true, false},
		{"Read pod metrics", userNS.Name, true, false},
		{"Start hardware probes", testutil.Namespace, true, false},
		{"Attach to pods", userNS.Name, false, true},
	}
	for _, c := range cases {
		p := byLabel[c.label]
		if p.Namespace != c.namespace || p.Allowed != c.allowed || p.unknown != c.unknown {
			t.Errorf("%s = %+v, want namespace %q, allowed %v, unknown %v",
				c.label, p, c.namespace, c.allowed, c.unknown)
		}
	}
}
