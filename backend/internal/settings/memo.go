package settings

import (
	"context"
	"errors"
	"sync"

	"app/internal/kube"
)

// memo keeps one value read from the API server until forget. Parallel callers share one read and its result,
// also a failure; a failed read is not kept, so the next caller reads again.
type memo[T any] struct {
	mu     sync.Mutex
	value  *T         // guarded by mu; nil: the next get reads
	flight *flight[T] // guarded by mu; the read running now, nil when none
}

// flight is one read; done is closed once value and err are set.
type flight[T any] struct {
	done  chan struct{}
	value T
	err   error
}

// get returns the kept value or reads it with load. load gets a context with the values of ctx that ends after
// kube.LoadTimeout, not when ctx ends: its result serves every caller waiting for it.
func (m *memo[T]) get(ctx context.Context, load func(context.Context) (T, error)) (T, error) {
	m.mu.Lock()
	if m.value != nil {
		v := *m.value
		m.mu.Unlock()
		return v, nil
	}
	f := m.flight
	if f == nil {
		f = &flight[T]{done: make(chan struct{})}
		m.flight = f
		m.mu.Unlock()
		m.run(ctx, f, load)
	} else {
		m.mu.Unlock()
	}
	select {
	case <-f.done:
		return f.value, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// run reads the value of a flight; a load that panics leaves errInterrupted to the waiting callers.
func (m *memo[T]) run(ctx context.Context, f *flight[T], load func(context.Context) (T, error)) {
	f.err = errInterrupted
	defer m.finish(f)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), kube.LoadTimeout)
	defer cancel()
	f.value, f.err = load(ctx)
}

var errInterrupted = errors.New("reading the settings was interrupted")

// finish keeps the value of a successful flight, unless forget was called meanwhile, and wakes its callers.
func (m *memo[T]) finish(f *flight[T]) {
	m.mu.Lock()
	if m.flight == f {
		m.flight = nil
		if f.err == nil {
			m.value = &f.value
		}
	}
	m.mu.Unlock()
	close(f.done)
}

// forget drops the kept value; a read still running serves its waiting callers but is not kept.
func (m *memo[T]) forget() {
	m.mu.Lock()
	m.value, m.flight = nil, nil
	m.mu.Unlock()
}
