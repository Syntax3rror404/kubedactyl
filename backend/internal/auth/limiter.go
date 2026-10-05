package auth

import (
	"slices"
	"sync"
	"time"
)

// Limiter throttles failed attempts per key (e.g. client IP + username). Its memory is bounded:
// expired keys are swept once per window, and at most DefaultMaxKeys keys are remembered.
type Limiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	maxKeys  int
	failures map[string][]time.Time
	swept    time.Time
}

// DefaultMaxKeys is how many keys a limiter remembers at most (many different client IPs or
// usernames must not fill the memory).
const DefaultMaxKeys = 10000

// NewLimiter allows max failures per key within window.
func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, maxKeys: DefaultMaxKeys, failures: map[string][]time.Time{}}
}

func (l *Limiter) recent(key string, now time.Time) []time.Time {
	var keep []time.Time
	for _, t := range l.failures[key] {
		if now.Sub(t) < l.window {
			keep = append(keep, t)
		}
	}
	if len(keep) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = keep
	}
	return keep
}

// Take counts an attempt before it is checked and reports whether it may run (if not: when to retry).
// Counting first means parallel attempts cannot all start before the first one has failed; Forgive takes
// back the attempt of a success.
func (l *Limiter) Take(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if recent := l.recent(key, now); len(recent) >= l.max {
		return false, l.window - now.Sub(recent[0])
	}
	if now.Sub(l.swept) >= l.window {
		l.sweep(now)
	}
	if _, known := l.failures[key]; !known && len(l.failures) >= l.maxKeys {
		l.sweep(now)
		if len(l.failures) >= l.maxKeys {
			l.evictOldest()
		}
	}
	l.failures[key] = append(l.recent(key, now), now)
	return true, 0
}

// Forgive takes back an attempt counted by Take at the time at.
func (l *Limiter) Forgive(key string, at time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	times := l.failures[key]
	if i := slices.Index(times, at); i >= 0 {
		times = slices.Delete(times, i, i+1)
	}
	if len(times) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = times
	}
}

// sweep drops every key whose failures are all older than the window.
func (l *Limiter) sweep(now time.Time) {
	l.swept = now
	for key := range l.failures {
		l.recent(key, now)
	}
}

// evictOldest drops the key whose last failure is the oldest (only when the limiter is full of
// keys that are all still within the window).
func (l *Limiter) evictOldest() {
	var oldest string
	var at time.Time
	found := false
	for key, times := range l.failures {
		if last := times[len(times)-1]; !found || last.Before(at) {
			oldest, at, found = key, last, true
		}
	}
	delete(l.failures, oldest)
}

// Reset clears the failures of a key after a successful login.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}
