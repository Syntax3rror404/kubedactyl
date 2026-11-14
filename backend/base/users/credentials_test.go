package users

import (
	"context"
	"errors"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/base/auth"
)

func TestUpdatePasswordHash(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password", Role: v1alpha1.DefaultRole})
	if err != nil {
		t.Fatal(err)
	}
	current := u.Spec.PasswordHash
	if err := s.UpdatePasswordHash(ctx, u, "a-long-password"); err != nil || u.Spec.PasswordHash != current {
		t.Fatalf("a current hash must stay: %v", err)
	}
	// A hash with the former parameters (64 MiB) is replaced; the session epoch stays.
	u.Spec.PasswordHash = strings.Replace(current, "m=19456,t=2,p=1", "m=65536,t=3,p=2", 1)
	if err := s.UpdatePasswordHash(ctx, u, "a-long-password"); err != nil {
		t.Fatal(err)
	}
	got := &v1alpha1.User{}
	if err := s.Client.Get(ctx, client.ObjectKeyFromObject(u), got); err != nil {
		t.Fatal(err)
	}
	if auth.Outdated(got.Spec.PasswordHash) || !auth.VerifyPassword("a-long-password", got.Spec.PasswordHash) ||
		got.Spec.SessionEpoch != u.Spec.SessionEpoch {
		t.Errorf("hash not replaced or sessions ended: %+v", got.Spec)
	}
}

// TestUpdatePasswordHashKeepsConcurrentChanges: a change made since the sign-in read the account wins.
func TestUpdatePasswordHashKeepsConcurrentChanges(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"})
	if err != nil {
		t.Fatal(err)
	}
	stale := u.DeepCopy()
	stale.Spec.PasswordHash = strings.Replace(u.Spec.PasswordHash, "m=19456,t=2,p=1", "m=65536,t=3,p=2", 1)
	if err := s.EndSessions(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePasswordHash(ctx, stale, "a-long-password"); err != nil {
		t.Errorf("a conflict skips the new hash: %v", err)
	}
	got := get(t, s, "alice")
	if got.Spec.SessionEpoch != stale.Spec.SessionEpoch+1 || got.Spec.PasswordHash != u.Spec.PasswordHash {
		t.Errorf("the concurrent change was overwritten: %+v", got.Spec)
	}
	if err := s.EndSessions(ctx, stale); !apierrors.IsConflict(err) {
		t.Errorf("ending sessions with a stale account: %v, want a conflict", err)
	}
}

func TestAddTokenKeepsTheLimit(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, err := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range MaxTokens {
		if err := s.AddToken(ctx, u, v1alpha1.APIToken{ID: string(rune('a' + i)), Name: "ci"}); err != nil {
			t.Fatalf("token %d: %v", i, err)
		}
	}
	if err := s.AddToken(ctx, u, v1alpha1.APIToken{ID: "over", Name: "ci"}); !errors.Is(err, ErrTokenLimit) {
		t.Errorf("token over the limit: %v", err)
	}
	if n := len(get(t, s, "alice").Spec.Tokens); n != MaxTokens {
		t.Errorf("%d tokens stored, want %d", n, MaxTokens)
	}
}
