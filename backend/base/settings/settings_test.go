package settings

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"app/api/v1alpha1"
	"app/base/kube"
	"app/base/validation"
)

const namespace = "panel"

// newClient is a fake cluster with the panel's resources and objs.
func newClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func TestKubeAPIQPS(t *testing.T) {
	ctx := t.Context()
	c := newClient(t)
	limiter := kube.NewRateLimiter(DefaultKubeAPIQPS, DefaultKubeAPIUserQPS)
	spec := func(qps int32) v1alpha1.PanelSettingsSpec {
		return v1alpha1.PanelSettingsSpec{KubeAPIQPS: qps}
	}
	s := &Store{Client: c, Reader: c, Namespace: namespace, Limiter: limiter}

	if spec, err := s.Get(ctx); err != nil || spec.KubeAPIQPS != DefaultKubeAPIQPS {
		t.Fatalf("default: %d %v", spec.KubeAPIQPS, err)
	}
	for _, bad := range []int32{4, 1001} {
		if _, err := s.Update(ctx, spec(bad)); err == nil {
			t.Errorf("%d requests per second accepted", bad)
		}
	}
	if _, err := s.Update(ctx, spec(120)); err != nil {
		t.Fatal(err)
	}
	if limiter.QPS() != 120 {
		t.Errorf("saved limit not applied: %v", limiter.QPS())
	}
	if spec, _ := s.Get(ctx); spec.KubeAPIQPS != 120 {
		t.Errorf("stored limit: %d", spec.KubeAPIQPS)
	}
}

// countingReader counts the reads that would reach the API server.
type countingReader struct {
	client.Reader
	gets atomic.Int32
}

func (r *countingReader) Get(
	ctx context.Context, key client.ObjectKey, obj client.Object, o ...client.GetOption,
) error {
	r.gets.Add(1)
	return r.Reader.Get(ctx, key, obj, o...)
}

func TestCurrentKeepsSettingsUntilTheyChange(t *testing.T) {
	ctx := t.Context()
	obj := &v1alpha1.PanelSettings{
		ObjectMeta: metav1.ObjectMeta{Name: v1alpha1.SettingsName, Namespace: namespace},
		Spec:       v1alpha1.PanelSettingsSpec{BrandName: "Acme"},
	}
	c := newClient(t, obj)
	r := &countingReader{Reader: c}
	s := &Store{Client: c, Reader: r, Namespace: namespace}
	informers := &informertest.FakeInformers{Scheme: c.Scheme()}
	if err := s.Watch(ctx, informers); err != nil {
		t.Fatal(err)
	}
	brand := func() string {
		spec, err := s.Current(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return spec.BrandName
	}
	for range 5 {
		brand()
	}
	if n := r.gets.Load(); n != 1 {
		t.Fatalf("5 calls read the API server %d times, want 1", n)
	}

	// Changed with kubectl: the informer reports it, the next call reads once.
	changed := obj.DeepCopy()
	changed.Spec.BrandName = "Changed"
	if err := c.Update(ctx, changed); err != nil {
		t.Fatal(err)
	}
	if got := brand(); got != "Acme" {
		t.Fatalf("kept settings: %q", got)
	}
	inf, _ := informers.FakeInformerFor(ctx, &v1alpha1.PanelSettings{})
	other := changed.DeepCopy()
	other.Namespace = "another-installation"
	inf.Update(other, other)
	if got := brand(); got != "Acme" {
		t.Errorf("settings of another installation dropped the kept ones: %q", got)
	}
	inf.Update(obj, changed)
	if got := brand(); got != "Changed" || r.gets.Load() != 2 {
		t.Errorf("after the informer event: %q, %d reads", got, r.gets.Load())
	}

	// Saved through the store: applies at once.
	saved := v1alpha1.PanelSettingsSpec{BrandName: "Saved"}
	if _, err := s.Update(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if got := brand(); got != "Saved" {
		t.Errorf("after a save: %q", got)
	}
}

// heldReader holds every Get until release is closed, counts them and fails them with err (nil: reads).
type heldReader struct {
	client.Reader
	release chan struct{}
	err     error
	gets    atomic.Int32
}

func (r *heldReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, o ...client.GetOption) error {
	r.gets.Add(1)
	<-r.release
	if r.err != nil {
		return r.err
	}
	return r.Reader.Get(ctx, key, obj, o...)
}

// TestCurrentSharesOneRead: callers that arrive while the settings are read wait for that read and get its
// result, also a failure; a failure is not kept.
func TestCurrentSharesOneRead(t *testing.T) {
	for _, failure := range []error{nil, errors.New("api server unavailable")} {
		synctest.Test(t, func(t *testing.T) {
			c := newClient(t)
			r := &heldReader{Reader: c, release: make(chan struct{}), err: failure}
			s := &Store{Client: c, Reader: r, Namespace: namespace}
			errs := make(chan error, 5)
			for range 5 {
				go func() {
					_, err := s.Current(t.Context())
					errs <- err
				}()
			}
			synctest.Wait()
			close(r.release)
			for range 5 {
				if err := <-errs; !errors.Is(err, failure) {
					t.Errorf("caller got %v, want %v", err, failure)
				}
			}
			_, _ = s.Current(t.Context())
			if want := map[bool]int32{true: 1, false: 2}[failure == nil]; r.gets.Load() != want {
				t.Errorf("failure %v: %d reads, want %d", failure, r.gets.Load(), want)
			}
		})
	}
}

// TestValidateReportsEveryField: every wrong field is reported at once, so the form shows them together.
func TestValidateReportsEveryField(t *testing.T) {
	s := &Store{}
	spec := v1alpha1.PanelSettingsSpec{BrandLogo: "https://example.com/logo.png", Favicon: "bogus", SessionHours: -1}
	_, err := s.Validate(t.Context(), spec)
	var errs validation.Errors
	if !errors.As(err, &errs) || errs["brandLogo"] == "" || errs["favicon"] == "" || errs["sessionHours"] == "" {
		t.Fatalf("err = %v, want brandLogo, favicon and sessionHours", err)
	}
}

// Without settings (and for settings without lifetimes) the default lifetimes are shown.
func TestGetFillsInLifetimes(t *testing.T) {
	c := newClient(t)
	s := &Store{Client: c, Reader: c, Namespace: namespace}
	spec, err := s.Get(t.Context())
	if err != nil || spec.SessionHours != DefaultSessionHours || spec.APITokenMaxDays != DefaultAPITokenMaxDays {
		t.Fatalf("%+v %v", spec, err)
	}
}
