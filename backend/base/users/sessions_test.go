package users

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
)

// TestLogOut: a logged out session stays in the user's status until it would have expired; the next logout drops
// the expired ones.
func TestLogOut(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if _, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.LogOut(ctx, "alice", "old", now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.LogOut(ctx, "alice", "a", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	u := get(t, s, "alice")
	if !LoggedOut(u, "a") || LoggedOut(u, "b") {
		t.Errorf("logged out: %+v", u.Status.LoggedOut)
	}
	if len(u.Status.LoggedOut) != 1 {
		t.Errorf("the expired session must be dropped: %+v", u.Status.LoggedOut)
	}
}

// TestLogOutOverTheLimitEndsEverySession: with MaxLoggedOut kept, one more logout ends every session of the user
// (a new epoch) and starts the list again.
func TestLogOutOverTheLimitEndsEverySession(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"})
	if err != nil {
		t.Fatal(err)
	}
	expires := metav1.NewTime(time.Now().Add(time.Hour))
	for i := range MaxLoggedOut {
		entry := v1alpha1.LoggedOutSession{ID: fmt.Sprint(i), ExpiresAt: expires}
		u.Status.LoggedOut = append(u.Status.LoggedOut, entry)
	}
	if err := s.Client.Status().Update(ctx, u); err != nil {
		t.Fatal(err)
	}
	epoch := get(t, s, "alice").Spec.SessionEpoch
	if err := s.LogOut(ctx, "alice", "one-more", expires.Time); err != nil {
		t.Fatal(err)
	}
	got := get(t, s, "alice")
	if got.Spec.SessionEpoch != epoch+1 || len(got.Status.LoggedOut) != 0 {
		t.Errorf("epoch %d (was %d), %d logged out kept", got.Spec.SessionEpoch, epoch, len(got.Status.LoggedOut))
	}
}
