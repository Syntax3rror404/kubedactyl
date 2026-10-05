// Package users manages the panel accounts (User resources in the panel namespace): creating a
// user together with its namespace, applying changes, and the rule that at least one active
// administrator remains.
package users

import (
	"context"
	"errors"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/tenancy"
	"app/internal/validation"
)

// Errors of changes that would break the account rules.
var (
	ErrLastAdmin     = errors.New("the last administrator cannot be removed, demoted or disabled")
	ErrOwnAccount    = errors.New("you cannot delete your own account")
	ErrWrongPassword = errors.New("current password is wrong")
	ErrUsernameTaken = errors.New("this username is taken")
)

// Store creates and changes users.
type Store struct {
	Client client.Client
	// Reader reads uncached: the administrator count must not come from a lagging cache.
	Reader    client.Reader
	Namespace string

	// adminChanges serializes the changes that can remove an administrator: counting and
	// changing must not interleave, or two requests could each see "another admin exists" and
	// remove the last two. The panel runs as a single replica (Deployment strategy Recreate).
	adminChanges sync.Mutex
	// inviteUses serializes accepting invites, so a link creates one account only.
	inviteUses sync.Mutex
}

// CreateInput is a user to create; the role defaults to user. MustChangePassword makes the user replace the
// password after signing in.
type CreateInput struct {
	Username, Password, DisplayName, Email string
	Role                                   v1alpha1.UserRole
	MustChangePassword                     bool
}

// UpdateInput changes a user; nil fields stay unchanged. A new password or disabling the user
// ends all of its sessions; a new password also revokes its API tokens.
type UpdateInput struct {
	DisplayName        *string
	Email              *string
	Role               *v1alpha1.UserRole
	Disabled           *bool
	Password           *string
	MustChangePassword *bool
}

// Create validates the user, stores it and creates its namespace, so servers can be assigned
// right away. Invalid input is reported as validation.Errors.
func (s *Store) Create(ctx context.Context, in CreateInput) (*v1alpha1.User, error) {
	name := strings.ToLower(strings.TrimSpace(in.Username))
	if in.Role == "" {
		in.Role = v1alpha1.RoleUser
	}
	errs := validation.Errors{}
	if err := tenancy.ValidateUsername(name); err != nil {
		errs["username"] = err.Error()
	}
	if err := validRole(in.Role); err != nil {
		errs["role"] = err.Error()
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		errs["password"] = err.Error()
	}
	if err := errs.OrNil(); err != nil {
		return nil, err
	}
	u := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Namespace},
		Spec: v1alpha1.UserSpec{
			DisplayName: strings.TrimSpace(in.DisplayName), Email: strings.TrimSpace(in.Email),
			Role: in.Role, PasswordHash: hash, MustChangePassword: in.MustChangePassword,
		},
	}
	if err := s.Client.Create(ctx, u); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, validation.Field("username", ErrUsernameTaken)
		}
		return nil, err
	}
	if _, err := tenancy.EnsureNamespace(ctx, s.Client, name); err != nil {
		return nil, err
	}
	return u, nil
}

// Update applies the changes; the last active administrator can neither be demoted nor disabled.
func (s *Store) Update(ctx context.Context, u *v1alpha1.User, ch UpdateInput) error {
	if ch.removesAdmin(u) {
		s.adminChanges.Lock()
		defer s.adminChanges.Unlock()
		if err := s.keepOneAdmin(ctx); err != nil {
			return err
		}
	}
	patch := client.MergeFrom(u.DeepCopy())
	if err := ch.apply(&u.Spec); err != nil {
		return err
	}
	return s.Client.Patch(ctx, u, patch)
}

// UpdatePassword replaces the own password after checking the current one. Every session of
// the user ends (this one too); revokeTokens also removes the API tokens. A required password
// change is done with it.
func (s *Store) UpdatePassword(
	ctx context.Context, u *v1alpha1.User, current, next string, revokeTokens bool,
) error {
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
// Sessions stay valid.
func (s *Store) UpdatePasswordHash(ctx context.Context, u *v1alpha1.User, password string) error {
	if !auth.Outdated(u.Spec.PasswordHash) {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	patch := client.MergeFrom(u.DeepCopy())
	u.Spec.PasswordHash = hash
	return s.Client.Patch(ctx, u, patch)
}

// EndSessions ends every session of the user (a new session epoch). API tokens stay valid.
func (s *Store) EndSessions(ctx context.Context, u *v1alpha1.User) error {
	return s.endSessions(ctx, u, func(*v1alpha1.UserSpec) {})
}

func (s *Store) endSessions(ctx context.Context, u *v1alpha1.User, change func(*v1alpha1.UserSpec)) error {
	patch := client.MergeFrom(u.DeepCopy())
	change(&u.Spec)
	u.Spec.SessionEpoch++
	return s.Client.Patch(ctx, u, patch)
}

// Delete removes a user (the controller removes its namespace and servers). by is the caller.
func (s *Store) Delete(ctx context.Context, u *v1alpha1.User, by string) error {
	if u.Name == by {
		return ErrOwnAccount
	}
	if isActiveAdmin(u) {
		s.adminChanges.Lock()
		defer s.adminChanges.Unlock()
		if err := s.keepOneAdmin(ctx); err != nil {
			return err
		}
	}
	return s.Client.Delete(ctx, u)
}

// ActiveAdmins counts the enabled administrators (uncached).
func (s *Store) ActiveAdmins(ctx context.Context) (int, error) {
	return s.countAdmins(ctx, s.Reader)
}

// AdminCached reports whether the cache holds an enabled administrator (a check without a call to the API
// server).
func (s *Store) AdminCached(ctx context.Context) bool {
	n, err := s.countAdmins(ctx, s.Client)
	return err == nil && n > 0
}

func (s *Store) countAdmins(ctx context.Context, r client.Reader) (int, error) {
	var list v1alpha1.UserList
	if err := r.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return 0, err
	}
	n := 0
	for i := range list.Items {
		if isActiveAdmin(&list.Items[i]) {
			n++
		}
	}
	return n, nil
}

// keepOneAdmin fails unless another active administrator exists.
func (s *Store) keepOneAdmin(ctx context.Context) error {
	n, err := s.ActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func isActiveAdmin(u *v1alpha1.User) bool {
	return u.Spec.Role == v1alpha1.RoleAdmin && !u.Spec.Disabled && u.DeletionTimestamp == nil
}

func validRole(r v1alpha1.UserRole) error {
	if r != v1alpha1.RoleAdmin && r != v1alpha1.RoleUser {
		return errors.New("role must be admin or user")
	}
	return nil
}

func (ch UpdateInput) removesAdmin(u *v1alpha1.User) bool {
	if !isActiveAdmin(u) {
		return false
	}
	return (ch.Role != nil && *ch.Role != v1alpha1.RoleAdmin) || (ch.Disabled != nil && *ch.Disabled)
}

func (ch UpdateInput) apply(s *v1alpha1.UserSpec) error {
	if ch.DisplayName != nil {
		s.DisplayName = strings.TrimSpace(*ch.DisplayName)
	}
	if ch.Email != nil {
		s.Email = strings.TrimSpace(*ch.Email)
	}
	if ch.Role != nil {
		if err := validRole(*ch.Role); err != nil {
			return validation.Field("role", err)
		}
		s.Role = *ch.Role
	}
	if ch.Disabled != nil {
		s.Disabled = *ch.Disabled
		if *ch.Disabled {
			s.SessionEpoch++
		}
	}
	if ch.MustChangePassword != nil {
		s.MustChangePassword = *ch.MustChangePassword
	}
	if ch.Password != nil {
		hash, err := auth.HashPassword(*ch.Password)
		if err != nil {
			return validation.Field("password", err)
		}
		// A reset recovers the account: whoever had it keeps neither a session nor a token.
		s.PasswordHash = hash
		s.SessionEpoch++
		s.Tokens = nil
	}
	return nil
}
