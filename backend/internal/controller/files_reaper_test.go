package controller

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIdlePod(t *testing.T) {
	now := time.Now()
	pod := func(age time.Duration, readyFor time.Duration) *corev1.Pod {
		p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(now.Add(-age))}}
		if readyFor > 0 {
			p.Status.Conditions = []corev1.PodCondition{
				{
					Type:               corev1.PodReady,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(now.Add(-readyFor)),
				},
			}
		}
		return p
	}
	deleting := pod(time.Hour, time.Hour)
	deleting.DeletionTimestamp = &metav1.Time{Time: now}
	for _, tc := range []struct {
		name    string
		pod     *corev1.Pod
		lastUse time.Time
		want    bool
	}{
		// A volume that was just released can take over a minute to attach.
		{"still starting after two minutes", pod(2*time.Minute, 0), time.Time{}, false},
		{"stuck while starting", pod(11*time.Minute, 0), time.Time{}, true},
		{"ready only shortly after a slow start", pod(3*time.Minute, 20*time.Second), time.Time{}, false},
		{"idle since ready", pod(3*time.Minute, 2*time.Minute), time.Time{}, true},
		{"in use", pod(time.Hour, time.Hour), now.Add(-10 * time.Second), false},
		{"last used a minute ago", pod(time.Hour, time.Hour), now.Add(-time.Minute), true},
		{"already going", deleting, time.Time{}, false},
	} {
		if got := idlePod(tc.pod, tc.lastUse, now); got != tc.want {
			t.Errorf("%s: idle = %v, want %v", tc.name, got, tc.want)
		}
	}
}
