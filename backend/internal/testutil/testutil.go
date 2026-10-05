// Package testutil holds the setup the tests share: a fake Kubernetes client that knows the
// Kubernetes and panel types, the namespaces of a test installation and a silent logger.
package testutil

import (
	"log/slog"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"app/api/v1alpha1"
	"app/internal/tenancy"
)

// Namespace is the panel namespace of the test installation (the default of package tenancy); user namespaces are
// tenancy.Namespace("<user>") = "kubedactyl-user-<user>".
const Namespace = "kubedactyl"

// Builder returns a fake client builder with all types registered and sets the tenancy naming
// of the test installation. Add objects, indexes or status subresources and call Build().
func Builder(t testing.TB) *fake.ClientBuilder {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	tenancy.SystemNamespace, tenancy.NamespacePrefix = Namespace, Namespace+"-user-"
	return fake.NewClientBuilder().WithScheme(scheme)
}

// Logger discards everything.
func Logger() *slog.Logger { return slog.New(slog.DiscardHandler) }
