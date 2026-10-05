package kube

import (
	"errors"
	"testing"
	"time"
)

func TestRateLimiterChangesAtRuntime(t *testing.T) {
	r := NewRateLimiter(5, 10)
	accepted := 0
	for r.TryAccept() {
		accepted++
	}
	if accepted != 10 {
		t.Errorf("burst: %d requests, want 10 (twice the limit)", accepted)
	}
	r.SetLimits(1000, 10)
	start := time.Now()
	if err := r.Wait(t.Context()); err != nil || time.Since(start) > 100*time.Millisecond {
		t.Errorf("a higher limit must apply at once: %v after %v", err, time.Since(start))
	}
	if r.QPS() != 1000 {
		t.Errorf("QPS: %v", r.QPS())
	}
}

func TestAdmitLimitsEachUser(t *testing.T) {
	r := NewRateLimiter(100, 2)
	for i := range 4 { // burst: twice the rate
		if _, err := r.Admit("alice"); err != nil {
			t.Fatalf("request %d of alice: %v", i, err)
		}
	}
	wait, err := r.Admit("alice")
	if !errors.Is(err, ErrUserLimit) || wait <= 0 || wait > time.Second {
		t.Errorf("alice over her limit: %v, try again in %v", err, wait)
	}
	if _, err := r.Admit("bob"); err != nil {
		t.Errorf("bob must not notice alice: %v", err)
	}
	r.SetLimits(100, 50)
	time.Sleep(50 * time.Millisecond) // a higher rate refills faster at once
	if _, err := r.Admit("alice"); err != nil {
		t.Errorf("after raising the limit: %v", err)
	}
}

func TestAdmitRefusesWhileTheQueueIsLong(t *testing.T) {
	r := NewRateLimiter(5, 100)
	// 10 burst + 20 waiting calls = 4 s of queue at 5 per second.
	r.l.ReserveN(time.Now(), 10)
	r.l.ReserveN(time.Now(), 10)
	r.l.ReserveN(time.Now(), 10)
	if wait, err := r.Busy(); !errors.Is(err, ErrBusy) || wait <= 0 {
		t.Errorf("long queue: %v, try again in %v", err, wait)
	}
	r.SetLimits(1000, 100)
	if _, err := r.Busy(); err != nil {
		t.Errorf("a higher limit empties the queue at once: %v", err)
	}
}

func TestMeterAveragesTheLastFullSeconds(t *testing.T) {
	var m meter
	now := time.Unix(1000, 0)
	for i := range 20 { // 4 per second for 5 seconds
		m.add(now.Add(time.Duration(i) * 250 * time.Millisecond))
	}
	m.add(now.Add(5 * time.Second)) // still counting, not part of the average
	if got := m.rate(now.Add(5 * time.Second)); got != 4 {
		t.Errorf("rate: %v, want 4", got)
	}
	if got := m.rate(now.Add(11 * time.Second)); got != 0 {
		t.Errorf("rate after a quiet while: %v, want 0", got)
	}
}

func TestRatesCountRefusedRequestsAndWaitingCalls(t *testing.T) {
	r := NewRateLimiter(50, 2)
	for range 6 { // 4 admitted, 2 refused
		_, _ = r.Admit("alice")
	}
	r.TryAccept()
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
	if got := r.UserRate("alice"); got.Rate != 6.0/meterSeconds || got.Limit != 2 {
		t.Errorf("alice: %+v", got)
	}
	if got := r.UserRate("bob"); got.Rate != 0 {
		t.Errorf("bob: %+v", got)
	}
	if got := r.PanelRate(); got.Rate != 1.0/meterSeconds || got.Limit != 50 {
		t.Errorf("panel: %+v", got)
	}
}
