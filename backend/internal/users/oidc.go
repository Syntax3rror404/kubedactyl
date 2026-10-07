package users

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/sso"
	"app/internal/tenancy"
)

// Errors of a sign-in through the identity provider.
var (
	ErrDisabled = errors.New("this account is disabled")
	// ErrOtherIdentity: the account with the username is linked to another user of the identity provider.
	ErrOtherIdentity = errors.New("the account is linked to another user of the identity provider")
	// ErrNotLinked: an account with the username exists but is not linked, and linking by username is off.
	ErrNotLinked = errors.New("an account with this username exists and is not linked to the identity provider")
	// ErrInvalidUsername: the username claim is no valid panel username.
	ErrInvalidUsername = errors.New("the username from the identity provider is not a valid panel username")
)

// SignInOIDC returns the account of a user the identity provider signed in: the account linked to that user,
// else (with LinkByUsername) the account with the same username, linked now, else a new one. Display name,
// email and role come from the identity provider; without KeepPasswords a linked account loses its password
// and signs in only through the identity provider.
func (s *Store) SignInOIDC(ctx context.Context, acc sso.Account, o v1alpha1.OIDCSettings) (*v1alpha1.User, error) {
	u, isNew, err := s.findAccount(ctx, acc, o.LinkByUsername)
	if err != nil {
		return nil, err
	}
	if isNew {
		link(&u.Spec, acc, o.KeepPasswords)
		return u, s.create(ctx, u)
	}
	if u.Spec.Disabled {
		return nil, ErrDisabled
	}
	removesAdmin := isActiveAdmin(u) && acc.Role != v1alpha1.RoleAdmin
	return u, s.change(ctx, u, removesAdmin, func(spec *v1alpha1.UserSpec) error {
		link(spec, acc, o.KeepPasswords)
		return nil
	})
}

// link takes the profile and role from the identity provider and links the account to its user. Linking an
// existing account ends its sessions and API tokens: they were made before the account belonged to that user.
func link(spec *v1alpha1.UserSpec, acc sso.Account, keepPassword bool) {
	if spec.OIDC == nil {
		spec.SessionEpoch++
		spec.Tokens = nil
	}
	spec.DisplayName, spec.Email, spec.Role = acc.DisplayName, acc.Email, acc.Role
	spec.OIDC = &acc.Identity
	if !keepPassword && spec.PasswordHash != "" {
		spec.PasswordHash, spec.MustChangePassword = "", false
	}
}

// findAccount finds the account of a user of the identity provider (the account is read uncached: it may have
// been linked a moment ago) or returns a new one, not stored yet. An account that is not linked is used only
// with linkByUsername.
func (s *Store) findAccount(
	ctx context.Context, acc sso.Account, linkByUsername bool,
) (u *v1alpha1.User, isNew bool, err error) {
	name, err := s.linkedTo(ctx, acc.Identity)
	if err != nil {
		return nil, false, err
	}
	if name == "" {
		name = acc.Username
		if err := tenancy.ValidateUsername(name); err != nil {
			return nil, false, fmt.Errorf("%w: %q", ErrInvalidUsername, name)
		}
	}
	u = &v1alpha1.User{}
	err = s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: name}, u)
	if apierrors.IsNotFound(err) {
		return s.newUser(name, v1alpha1.UserSpec{}), true, nil
	}
	if err != nil {
		return nil, false, err
	}
	switch {
	case u.Spec.OIDC != nil && *u.Spec.OIDC != acc.Identity:
		return nil, false, ErrOtherIdentity
	case u.Spec.OIDC == nil && !linkByUsername:
		return nil, false, ErrNotLinked
	}
	return u, false, nil
}

// linkedTo returns the name of the account linked to the identity ("" when none), from the cache: a user keeps
// the account when their username at the identity provider changes.
func (s *Store) linkedTo(ctx context.Context, id v1alpha1.OIDCIdentity) (string, error) {
	var list v1alpha1.UserList
	if err := s.Client.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return "", err
	}
	for i := range list.Items {
		if o := list.Items[i].Spec.OIDC; o != nil && *o == id {
			return list.Items[i].Name, nil
		}
	}
	return "", nil
}
