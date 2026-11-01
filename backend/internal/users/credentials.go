package users

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/validation"
)

// MaxTokens is how many API tokens an account may have.
const MaxTokens = 50

// Errors of signing in and of credentials.
var (
	// ErrDisabled means the account cannot sign in: it is disabled or being deleted, or the owner of a subuser is.
	ErrDisabled   = errors.New("this account is disabled")
	ErrTokenLimit = fmt.Errorf("you have the most API tokens you may have (%d), revoke one first", MaxTokens)
)

// CheckActive returns ErrDisabled for an account that may neither sign in nor use its sessions and tokens:
// disabled, being deleted, or a subuser whose owner is (or is gone). The owner is read from the cache.
func (s *Store) CheckActive(ctx context.Context, u *v1alpha1.User) error {
	if u.Spec.Disabled || u.DeletionTimestamp != nil {
		return ErrDisabled
	}
	if !Subuser(u) {
		return nil
	}
	owner := &v1alpha1.User{}
	err := s.Client.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: u.Spec.Owner}, owner)
	if apierrors.IsNotFound(err) || (err == nil && (owner.Spec.Disabled || owner.DeletionTimestamp != nil)) {
		return ErrDisabled
	}
	return err
}

// AddToken stores a new API token of the account, at most MaxTokens (ErrTokenLimit). Like every change, it fails
// when the account changed since u was read.
func (s *Store) AddToken(ctx context.Context, u *v1alpha1.User, t v1alpha1.APIToken) error {
	if len(u.Spec.Tokens) >= MaxTokens {
		return ErrTokenLimit
	}
	return s.change(ctx, u, false, func(spec *v1alpha1.UserSpec) error {
		spec.Tokens = append(spec.Tokens, t)
		return nil
	})
}

// UpdatePassword replaces the own password after checking the current one. Every session of
// the user ends (this one too); revokeTokens also removes the API tokens. A required password
// change is done with it.
func (s *Store) UpdatePassword(
	ctx context.Context, u *v1alpha1.User, current, next string, revokeTokens bool,
) error {
	if u.Spec.PasswordHash == "" {
		return ErrNoPassword
	}
	if !auth.VerifyPassword(current, u.Spec.PasswordHash) {
		return validation.Field("current", ErrWrongPassword)
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return validation.Field("new", err)
	}
	return s.endSessions(ctx, u, func(spec *v1alpha1.UserSpec) {
		spec.PasswordHash = hash
		spec.MustChangePassword = false
		if revokeTokens {
			spec.Tokens = nil
		}
	})
}

// UpdatePasswordHash hashes a password that was just verified again when its stored hash uses
// older parameters (auth.Outdated), so every check costs what the current parameters cost.
// Sessions stay valid; after a concurrent change of the account the old hash is kept.
func (s *Store) UpdatePasswordHash(ctx context.Context, u *v1alpha1.User, password string) error {
	if !auth.Outdated(u.Spec.PasswordHash) {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	patch := client.MergeFromWithOptions(u.DeepCopy(), client.MergeFromWithOptimisticLock{})
	u.Spec.PasswordHash = hash
	// A concurrent change wins: the hash is replaced at a later sign-in.
	if err := s.Client.Patch(ctx, u, patch); !apierrors.IsConflict(err) {
		return err
	}
	return nil
}

// EndSessions ends every session of the user (a new session epoch). API tokens stay valid. Like every change, it
// fails when the account changed since u was read.
func (s *Store) EndSessions(ctx context.Context, u *v1alpha1.User) error {
	return s.endSessions(ctx, u, func(*v1alpha1.UserSpec) {})
}

func (s *Store) endSessions(ctx context.Context, u *v1alpha1.User, change func(*v1alpha1.UserSpec)) error {
	return s.change(ctx, u, false, func(spec *v1alpha1.UserSpec) error {
		change(spec)
		spec.SessionEpoch++
		return nil
	})
}
