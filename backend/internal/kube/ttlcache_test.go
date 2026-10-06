package kube

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTTLCacheLoadsOncePerKey(t *testing.T) {
	c := NewTTLCache[string, int](time.Minute)
	var loads atomic.Int32
	load := func() (int, error) {
		loads.Add(1)
		time.Sleep(20 * time.Millisecond) // the others arrive while it loads
		return 42, nil
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if v, err := c.Get("a", load); v != 42 || err != nil {
				t.Errorf("got %d %v", v, err)
			}
		})
	}
	wg.Wait()
	if n := loads.Load(); n != 1 {
		t.Errorf("20 parallel requests loaded %d times, want 1", n)
	}
	if _, _ = c.Get("b", load); loads.Load() != 2 {
		t.Error("another key loads on its own")
	}
}

func TestTTLCacheExpiresAndKeepsNoErrors(t *testing.T) {
	c := NewTTLCache[string, int](20 * time.Millisecond)
	if _, err := c.Get("a", func() (int, error) { return 0, errors.New("down") }); err == nil {
		t.Fatal("the error must be returned")
	}
	if v, _ := c.Get("a", func() (int, error) { return 1, nil }); v != 1 {
		t.Error("a failed load must not be kept")
	}
	if v, _ := c.Get("a", func() (int, error) { return 2, nil }); v != 1 {
		t.Error("a fresh value must be kept")
	}
	time.Sleep(30 * time.Millisecond)
	if v, _ := c.Get("a", func() (int, error) { return 3, nil }); v != 3 {
		t.Error("an expired value must be loaded again")
	}
	time.Sleep(30 * time.Millisecond)
	_, _ = c.Get("b", func() (int, error) { return 0, nil })
	if _, ok := c.entries["a"]; ok {
		t.Error("expired entries must be dropped")
	}
}

func TestTTLCacheForget(t *testing.T) {
	c := NewTTLCache[string, int](time.Minute)
	_, _ = c.Get("a", func() (int, error) { return 1, nil })
	_, _ = c.Get("b", func() (int, error) { return 1, nil })
	c.Forget("a")
	if v, _ := c.Get("a", func() (int, error) { return 2, nil }); v != 2 {
		t.Error("a forgotten value must be loaded again")
	}
	if v, _ := c.Get("b", func() (int, error) { return 2, nil }); v != 1 {
		t.Error("other keys must be kept")
	}
}
