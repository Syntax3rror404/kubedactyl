package kube

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// MaxQueue is how long the Kubernetes API calls already waiting for the limit may take before
// new requests are refused (ErrBusy): the queue stays short instead of growing without end.
const MaxQueue = 3 * time.Second

// Why Admit refuses a request.
var (
	ErrUserLimit = errors.New("too many requests, wait a moment")
	ErrBusy      = errors.New("the panel is busy: the Kubernetes API limit is reached, try again in a moment")
)

// RateLimiter limits the requests the panel sends to the Kubernetes API. Every client of the
// panel shares it (rest.Config.RateLimiter), and the limits can change while the panel runs
// (panel settings). Busy keeps the queue short, and Admit limits the requests of each user, so
// one user cannot use up the limit of everybody. PanelRate and UserRate show how close both are.
type RateLimiter struct {
	l     *rate.Limiter
	calls meter

	mu      sync.Mutex
	userQPS int
	users   map[string]*user
}

// user is the limit and the request count of one user.
type user struct {
	l        *rate.Limiter
	requests meter
}

// Rate is how many requests per second were sent over the last seconds, and their limit.
type Rate struct {
	Rate  float64 `json:"rate"  example:"2.4"`
	Limit int     `json:"limit" example:"10"`
}

// NewRateLimiter allows qps requests per second to the Kubernetes API and userQPS requests per
// second of each user; both start with a full burst (twice the rate).
func NewRateLimiter(qps, userQPS int) *RateLimiter {
	return &RateLimiter{
		l: rate.NewLimiter(rate.Limit(qps), 2*qps), userQPS: userQPS, users: map[string]*user{},
	}
}

// SetLimits changes both rates at once (bursts of twice the rate).
func (r *RateLimiter) SetLimits(qps, userQPS int) {
	r.l.SetLimit(rate.Limit(qps))
	r.l.SetBurst(2 * qps)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userQPS = userQPS
	for _, u := range r.users {
		u.l.SetLimit(rate.Limit(userQPS))
		u.l.SetBurst(2 * userQPS)
	}
}

// Busy returns ErrBusy while the calls already waiting for the limit take longer than MaxQueue,
// and when to try again. Checked before a request starts, never in the middle, so no change is
// left half done.
func (r *RateLimiter) Busy() (time.Duration, error) {
	if wait := delay(r.l, time.Now()); wait > MaxQueue {
		return wait - MaxQueue, ErrBusy
	}
	return 0, nil
}

// Admit counts a request of name (it may lead to Kubernetes API calls): ErrUserLimit when the
// user sent too many, with when to try again.
func (r *RateLimiter) Admit(name string) (time.Duration, error) {
	now := time.Now()
	u := r.user(name)
	u.requests.add(now)
	if !u.l.AllowN(now, 1) {
		return delay(u.l, now), ErrUserLimit
	}
	return 0, nil
}

// UserRate is how many requests name sent per second (refused ones included) and the user limit.
func (r *RateLimiter) UserRate(name string) Rate {
	u := r.user(name)
	return Rate{Rate: u.requests.rate(time.Now()), Limit: int(u.l.Limit())}
}

// PanelRate is how many Kubernetes API calls the whole panel asked for per second (waiting ones
// included) and the panel limit.
func (r *RateLimiter) PanelRate() Rate {
	return Rate{Rate: r.calls.rate(time.Now()), Limit: int(r.l.Limit())}
}

func (r *RateLimiter) user(name string) *user {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[name]
	if !ok {
		u = &user{l: rate.NewLimiter(rate.Limit(r.userQPS), 2*r.userQPS)}
		r.users[name] = u
	}
	return u
}

// delay is how long a new request would wait for its token.
func delay(l *rate.Limiter, now time.Time) time.Duration {
	missing := 1 - l.TokensAt(now)
	if missing <= 0 {
		return 0
	}
	return time.Duration(missing / float64(l.Limit()) * float64(time.Second))
}

// meterSeconds is how many full seconds a meter averages over.
const meterSeconds = 5

// meter counts events per second.
type meter struct {
	mu    sync.Mutex
	slots [meterSeconds + 1]struct{ sec, n int64 }
}

func (m *meter) add(now time.Time) {
	sec := now.Unix()
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &m.slots[sec%int64(len(m.slots))]
	if s.sec != sec {
		s.sec, s.n = sec, 0
	}
	s.n++
}

// rate is the average per second over the last full seconds (the current one is still counting).
func (m *meter) rate(now time.Time) float64 {
	sec := now.Unix()
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, s := range m.slots {
		if s.sec >= sec-meterSeconds && s.sec < sec {
			n += s.n
		}
	}
	return float64(n) / meterSeconds
}

// The methods below implement flowcontrol.RateLimiter; every call is counted for PanelRate.

func (r *RateLimiter) TryAccept() bool {
	r.calls.add(time.Now())
	return r.l.Allow()
}

func (r *RateLimiter) Accept() {
	r.calls.add(time.Now())
	_ = r.l.Wait(context.Background())
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	r.calls.add(time.Now())
	return r.l.Wait(ctx)
}

func (r *RateLimiter) Stop()        {}
func (r *RateLimiter) QPS() float32 { return float32(r.l.Limit()) }
