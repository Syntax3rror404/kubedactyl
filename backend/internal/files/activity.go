package files

import (
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// IdleTimeout is how long a files pod stays after the last file operation.
const IdleTimeout = time.Minute

// Activity records when the files of a server were last used. The files pod of a server
// only exists while it is used; the controller removes it after IdleTimeout.
type Activity struct {
	mu   sync.Mutex
	last map[Ref]time.Time
}

// NewActivity returns an empty tracker.
func NewActivity() *Activity { return &Activity{last: map[Ref]time.Time{}} }

// Touch marks the files of a server as used now.
func (a *Activity) Touch(server Ref) {
	a.mu.Lock()
	a.last[server] = time.Now()
	a.mu.Unlock()
}

// Last returns when the files of a server were last used (zero: not since the panel started).
func (a *Activity) Last(server Ref) time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last[server]
}

// StopsAt is when an unused files pod is removed: IdleTimeout after the last file operation,
// counted from when the pod became ready at the earliest (the reaper checks every few seconds).
func StopsAt(lastUse, readySince time.Time) time.Time {
	if lastUse.Before(readySince) {
		lastUse = readySince
	}
	return lastUse.Add(IdleTimeout)
}

// ReadySince returns when a pod became ready (zero while it is not ready).
func ReadySince(pod *corev1.Pod) time.Time {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return c.LastTransitionTime.Time
		}
	}
	return time.Time{}
}

// Active reports whether the files were used within IdleTimeout.
func (a *Activity) Active(server Ref) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return time.Since(a.last[server]) < IdleTimeout
}

// Forget drops a server (after its files pod was removed).
func (a *Activity) Forget(server Ref) {
	a.mu.Lock()
	delete(a.last, server)
	a.mu.Unlock()
}
