package users

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/auth"
	"app/base/validation"
)

// UserResource and inviteResource name the resources in not found errors.
var (
	UserResource   = schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "users"}
	inviteResource = schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "invites"}
)

// InviteLifetime is how long an invite link works.
const InviteLifetime = 7 * 24 * time.Hour

// ErrInvalidInvite is returned for an invite link that is unknown, used or expired.
var ErrInvalidInvite = errors.New("the invite link is invalid or has expired, ask for a new one")

// InviteInput is an invite to create; an empty username lets the invited person choose it. Reset creates a link
// for a new password of the account Username. InviteRequest is the app's part (App.PrepareInvite).
type InviteInput struct {
	Reset    bool
	Username string
	Role     v1alpha1.UserRole
	Note     string
	app.InviteRequest
}

// AcceptInput is the account an invited person creates; Username is ignored when the invite names one.
type AcceptInput struct {
	Username, DisplayName, Password string
}

// CreateInvite stores a new invite of by and returns it together with the token of its link, which is not
// stored. A reset link is only for an account by manages.
func (s *Store) CreateInvite(
	ctx context.Context, in InviteInput, by *v1alpha1.User,
) (*v1alpha1.Invite, string, error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Role = cmp.Or(in.Role, v1alpha1.DefaultRole)
	target, err := s.checkInvite(ctx, in, by)
	if err != nil {
		return nil, "", err
	}
	s.inviteUses.Lock()
	defer s.inviteUses.Unlock()
	id := auth.NewInviteID()
	token, hash := auth.NewInviteToken(id)
	invite := &v1alpha1.Invite{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: s.Namespace},
		Spec: v1alpha1.InviteSpec{
			Username: in.Username, Role: in.Role, Note: strings.TrimSpace(in.Note), Hash: hash,
			ExpiresAt: metav1.NewTime(time.Now().Add(InviteLifetime)), CreatedBy: by.Name, Reset: in.Reset,
		},
	}
	if in.Reset {
		OwnedBy(&invite.ObjectMeta, target)
	}
	if s.App != nil {
		if err := s.App.PrepareInvite(ctx, invite, in.InviteRequest, by); err != nil {
			return nil, "", err
		}
	}
	if err := s.Client.Create(ctx, invite); err != nil {
		return nil, "", err
	}
	return invite, token, nil
}

// checkInvite validates an invite; for a password reset it returns the account.
func (s *Store) checkInvite(ctx context.Context, in InviteInput, by *v1alpha1.User) (*v1alpha1.User, error) {
	if in.Reset {
		return s.resetTarget(ctx, in.Username, by)
	}
	return nil, s.validateInvite(ctx, in)
}

// resetTarget reads an account that may get a password reset link: a local one (the identity provider manages
// linked ones) that by manages.
func (s *Store) resetTarget(ctx context.Context, name string, by *v1alpha1.User) (*v1alpha1.User, error) {
	if name == "" {
		return nil, validation.Field("username", errors.New("name the account"))
	}
	u := &v1alpha1.User{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: name}, u)
	if err == nil && !s.Manages(by, u) {
		err = apierrors.NewNotFound(UserResource, name)
	}
	if err != nil {
		return nil, err
	}
	if u.Spec.OIDC != nil {
		return nil, ErrManaged
	}
	return u, nil
}

func (s *Store) validateInvite(ctx context.Context, in InviteInput) error {
	errs := validation.Errors{}
	if err := validRole(in.Role); err != nil {
		errs["role"] = err.Error()
	}
	if in.Username != "" {
		if err := s.usernameFree(ctx, in.Username); err != nil {
			errs["username"] = err.Error()
		}
	}
	return errs.OrNil()
}

// usernameFree checks that a username is valid and not taken yet.
func (s *Store) usernameFree(ctx context.Context, name string) error {
	if err := ValidateUsername(name); err != nil {
		return err
	}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: name}, &v1alpha1.User{})
	if err == nil {
		return ErrUsernameTaken
	}
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// ListInvites returns the invites by manages, newest first, also expired ones. It reads uncached, so an invite
// created a moment ago is listed.
func (s *Store) ListInvites(ctx context.Context, by *v1alpha1.User) ([]v1alpha1.Invite, error) {
	var list v1alpha1.InviteList
	if err := s.Reader.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return nil, err
	}
	list.Items = slices.DeleteFunc(list.Items, func(i v1alpha1.Invite) bool { return !s.managesInvite(by, &i) })
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})
	return list.Items, nil
}

// RenewInvite gives an invite a new link, valid for the full lifetime again; the old link stops working. Expired
// invites can be renewed, too. Invites by does not manage are not found. It fails when the invite changed or was
// used meanwhile.
func (s *Store) RenewInvite(ctx context.Context, id string, by *v1alpha1.User) (*v1alpha1.Invite, string, error) {
	invite, err := s.getInvite(ctx, id, by)
	if err != nil {
		return nil, "", err
	}
	token, hash := auth.NewInviteToken(id)
	patch := client.MergeFromWithOptions(invite.DeepCopy(), client.MergeFromWithOptimisticLock{})
	invite.Spec.Hash = hash
	invite.Spec.ExpiresAt = metav1.NewTime(time.Now().Add(InviteLifetime))
	if err := s.Client.Patch(ctx, invite, patch); err != nil {
		return nil, "", err
	}
	return invite, token, nil
}

// DeleteInvite revokes an invite; its link stops working. Invites by does not manage are not found.
func (s *Store) DeleteInvite(ctx context.Context, id string, by *v1alpha1.User) error {
	invite, err := s.getInvite(ctx, id, by)
	if err != nil {
		return err
	}
	return s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID})
}

// getInvite reads an invite (uncached); invites by does not manage are not found.
func (s *Store) getInvite(ctx context.Context, id string, by *v1alpha1.User) (*v1alpha1.Invite, error) {
	invite := &v1alpha1.Invite{}
	if err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: id}, invite); err != nil {
		return nil, err
	}
	if !s.managesInvite(by, invite) {
		return nil, apierrors.NewNotFound(inviteResource, id)
	}
	return invite, nil
}

// managesInvite reports whether by may manage the invite: administrators every one, others the ones they own.
func (s *Store) managesInvite(by *v1alpha1.User, i *v1alpha1.Invite) bool {
	if by.Spec.Role == v1alpha1.RoleAdmin {
		return true
	}
	return s.App != nil && s.App.InviteOwner(i) != "" && s.App.InviteOwner(i) == by.Name
}

// FindInvite returns the invite of a valid link. It reads uncached: the link may be seconds old.
func (s *Store) FindInvite(ctx context.Context, token string) (*v1alpha1.Invite, error) {
	id := auth.InviteID(token)
	if id == "" {
		return nil, ErrInvalidInvite
	}
	invite := &v1alpha1.Invite{}
	if err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: id}, invite); err != nil {
		if apierrors.IsNotFound(err) || apierrors.IsInvalid(err) {
			return nil, ErrInvalidInvite
		}
		return nil, err
	}
	if !auth.MatchToken(token, invite.Spec.Hash) || !time.Now().Before(invite.Spec.ExpiresAt.Time) {
		return nil, ErrInvalidInvite
	}
	return invite, nil
}

// AcceptInvite creates the account of an invite link and removes the invite, so the link works once. It returns
// the account and the invite.
//  1. Account: created (or, for a reset link, given the new password).
//  2. Invite: deleted once the account exists, so the link is spent even when sharing fails.
//  3. App: what the invite grants besides the account (App.Accepted).
func (s *Store) AcceptInvite(
	ctx context.Context, token string, in AcceptInput,
) (*v1alpha1.User, *v1alpha1.Invite, error) {
	// Two requests with the same link must not both create an account: the second one finds no invite.
	s.inviteUses.Lock()
	defer s.inviteUses.Unlock()
	invite, err := s.FindInvite(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	if invite.Spec.Username != "" {
		in.Username = invite.Spec.Username
	}
	if invite.Spec.Reset {
		u, err := s.resetPassword(ctx, invite, in.Password)
		return u, invite, err
	}
	// 1. Account
	u, err := s.newAccount(CreateInput{
		Username: in.Username, Password: in.Password, DisplayName: in.DisplayName, Role: invite.Spec.Role,
	})
	if err != nil {
		return nil, nil, err
	}
	if s.App != nil {
		if err := s.App.Accept(ctx, invite, u); err != nil {
			return nil, nil, err
		}
	}
	if err := s.create(ctx, u); err != nil {
		return nil, nil, err
	}
	// 2. Invite
	if err := s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID}); err != nil {
		return nil, nil, err
	}
	// 3. App
	if s.App == nil {
		return u, invite, nil
	}
	return u, invite, s.App.Accepted(ctx, invite, u)
}

// resetPassword sets the password of a reset link's account and removes the link. Like a reset by an administrator,
// it ends every session and API token of the account. Accounts linked to the identity provider, disabled ones
// and those being deleted make the link invalid.
func (s *Store) resetPassword(ctx context.Context, invite *v1alpha1.Invite, password string) (*v1alpha1.User, error) {
	u := &v1alpha1.User{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: invite.Spec.Username}, u)
	if err == nil && u.Spec.OIDC != nil {
		return nil, ErrInvalidInvite
	}
	if err == nil {
		err = s.CheckActive(ctx, u)
	}
	if apierrors.IsNotFound(err) || errors.Is(err, ErrDisabled) {
		return nil, ErrInvalidInvite
	}
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, validation.Field("password", err)
	}
	err = s.endSessions(ctx, u, func(spec *v1alpha1.UserSpec) {
		spec.PasswordHash, spec.MustChangePassword, spec.Tokens = hash, false, nil
	})
	if err != nil {
		return nil, err
	}
	return u, s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID})
}

// deleteResetLinks deletes the reset links of an account (it was linked to the identity provider).
func (s *Store) deleteResetLinks(ctx context.Context, user string) error {
	var list v1alpha1.InviteList
	if err := s.Reader.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return err
	}
	for i := range list.Items {
		invite := &list.Items[i]
		if !invite.Spec.Reset || invite.Spec.Username != user {
			continue
		}
		err := s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID})
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}
