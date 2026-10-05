package files

import (
	"testing"
	"time"
)

func TestActivity(t *testing.T) {
	a := NewActivity()
	ref := Ref{Namespace: "ns", Name: "srv"}
	if a.Active(ref) {
		t.Error("unknown server must be idle")
	}
	a.Touch(ref)
	if !a.Active(ref) {
		t.Error("touched server must be active")
	}
	a.last[ref] = time.Now().Add(-IdleTimeout - time.Second)
	if a.Active(ref) {
		t.Error("server must be idle after the timeout")
	}
	a.Touch(ref)
	a.Forget(ref)
	if a.Active(ref) {
		t.Error("forgotten server must be idle")
	}
}

func TestStopsAt(t *testing.T) {
	now := time.Now()
	if got := StopsAt(time.Time{}, now); !got.Equal(now.Add(IdleTimeout)) {
		t.Errorf("never used: a minute after ready, got %v", got.Sub(now))
	}
	if got := StopsAt(now.Add(30*time.Second), now); !got.Equal(now.Add(30*time.Second + IdleTimeout)) {
		t.Errorf("used after ready: a minute after the last use, got %v", got.Sub(now))
	}
	if got := StopsAt(now.Add(-time.Hour), now); !got.Equal(now.Add(IdleTimeout)) {
		t.Errorf("used before it became ready: a minute after ready, got %v", got.Sub(now))
	}
}
