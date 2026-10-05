package kube

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// TestCacheLatencyIntegration measures how long new objects take to show up in a
// controller-runtime cache configured like the panel (KUBE_IT_CONTEXT required).
// KUBE_IT_HTTP2=1 uses HTTP/2 instead of HTTP/1.1 for comparison.
func TestCacheLatencyIntegration(t *testing.T) {
	kctx := os.Getenv("KUBE_IT_CONTEXT")
	if kctx == "" {
		t.Skip("KUBE_IT_CONTEXT not set")
	}
	cfg, err := config.GetConfigWithContext(kctx)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("KUBE_IT_HTTP2") == "" {
		cfg.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}
	ns := "kubedactyl-cachetest"
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	direct, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = direct.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
	defer func() {
		_ = direct.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
	}()

	c, err := cache.New(cfg, cache.Options{Scheme: scheme, DefaultNamespaces: map[string]cache.Config{ns: {}}})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = c.Start(ctx) }()
	if !c.WaitForCacheSync(ctx) {
		t.Fatal("cache did not sync")
	}
	// Start the informer by reading once.
	_ = c.List(ctx, &corev1.ConfigMapList{}, client.InNamespace(ns))

	var lat []time.Duration
	for i := 0; i < 18; i++ {
		name := fmt.Sprintf("probe-%d", i)
		start := time.Now()
		if err := direct.Create(
			ctx,
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}},
		); err != nil {
			t.Fatal(err)
		}
		for {
			if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &corev1.ConfigMap{}); err == nil {
				break
			}
			if time.Since(start) > 3*time.Minute {
				t.Fatalf("%s never appeared in the cache", name)
			}
			time.Sleep(50 * time.Millisecond)
		}
		lat = append(lat, time.Since(start))
		t.Logf("%s visible after %v", name, time.Since(start).Round(10*time.Millisecond))
		time.Sleep(5 * time.Second)
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	t.Logf("median %v, max %v", lat[len(lat)/2].Round(10*time.Millisecond), lat[len(lat)-1].Round(10*time.Millisecond))
}
