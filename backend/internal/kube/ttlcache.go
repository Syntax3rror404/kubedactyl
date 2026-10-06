package kube

import (
	"sync"
	"time"
)

// TTLCache keeps values read from the Kubernetes API for a fixed time, shared by every request: however many
// browsers ask, the API server is asked at most once per key and TTL. While a value loads, other requests for
// the same key wait for it instead of loading it again. Failed loads are not kept.
type TTLCache[K comparable, V any] struct {
	ttl time.Duration

	mu      sync.Mutex
	entries map[K]*ttlEntry[V]
	swept   time.Time
}

type ttlEntry[V any] struct {
	mu    sync.Mutex // held while the value loads
	value V
	at    time.Time
}

// NewTTLCache returns a cache that keeps each value for ttl.
func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{ttl: ttl, entries: map[K]*ttlEntry[V]{}}
}

// Get returns the value of key, calling load when it is missing or older than the TTL.
func (c *TTLCache[K, V]) Get(key K, load func() (V, error)) (V, error) {
	e := c.entry(key)
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.at.IsZero() && time.Since(e.at) < c.ttl {
		return e.value, nil
	}
	v, err := load()
	if err == nil {
		e.value, e.at = v, time.Now()
	}
	return v, err
}

// Forget drops the value of key after a change made through the panel, so the next Get loads it again. A load
// still running keeps its value to the requests already waiting for it, but it is not kept.
func (c *TTLCache[K, V]) Forget(key K) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// entry returns the entry of key; once per TTL it drops expired ones (servers that were deleted).
func (c *TTLCache[K, V]) entry(key K) *ttlEntry[V] {
	c.mu.Lock()
	defer c.mu.Unlock()
	if now := time.Now(); now.Sub(c.swept) >= c.ttl {
		c.swept = now
		for k, e := range c.entries {
			if e.mu.TryLock() {
				if now.Sub(e.at) >= c.ttl {
					delete(c.entries, k)
				}
				e.mu.Unlock()
			}
		}
	}
	e, ok := c.entries[key]
	if !ok {
		e = &ttlEntry[V]{}
		c.entries[key] = e
	}
	return e
}
