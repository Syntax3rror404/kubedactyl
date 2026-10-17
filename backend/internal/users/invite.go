package users

import (
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
	"app/internal/auth"
	"app/internal/tenancy"
	"app/internal/validation"
)

var inviteResource = schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "invites"}

// InviteLifetime is how long an invite link works.
const InviteLifetime = 7 * 24 * time.Hour

// ErrInvalidInvite is returned for an invite link that is unknown, used or expired.
var ErrInvalidInvite = errors.New("the invite link is invalid or has expired, ask for a new one")

// InviteInput is an invite to create; an empty username lets the invited person choose it. With Owner it invites
// a subuser of that user, who may have at most Limit subusers (accounts and invites) and sees the servers in Shares.
// Reset creates a link for a new password of the account Username (with Owner: a subuser of Owner).
type InviteInput struct {
	Reset    bool
	Username string
	Role     v1alpha1.UserRole
	Note     string
	Owner    string
	Limit    int
	Shares   []v1alpha1.InviteShare
}

// AcceptInput is the account an invited person creates; Username is ignored when the invite names one.
type AcceptInput struct {
	Username, DisplayName, Password string
}

// CreateInvite stores a new invite and returns it together with the token of its link, which is not stored.
func (s *Store) CreateInvite(ctx context.Context, in InviteInput, by string) (*v1alpha1.Invite, string, error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	if in.Role == "" || in.Owner != "" {
		in.Role = v1alpha1.RoleUser
	}
	if in.Owner == "" || in.Reset {
		in.Shares = nil
	}
	target, err := s.checkInvite(ctx, in)
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
			ExpiresAt: metav1.NewTime(time.Now().Add(InviteLifetime)), CreatedBy: by, Owner: in.Owner,
			Shares: in.Shares, Reset: in.Reset,
		},
	}
	switch {
	case in.Reset:
		ownedBy(&invite.ObjectMeta, target)
	case in.Owner != "":
		owner, err := s.subuserOwner(ctx, in.Owner, in.Limit)
		if err != nil {
			return nil, "", err
		}
		ownedBy(&invite.ObjectMeta, owner)
	}
	if err := s.Client.Create(ctx, invite); err != nil {
		return nil, "", err
	}
	return invite, token, nil
}

// checkInvite validates an invite; for a password reset it returns the account.
func (s *Store) checkInvite(ctx context.Context, in InviteInput) (*v1alpha1.User, error) {
	if in.Reset {
		return s.resetTarget(ctx, in.Username, in.Owner)
	}
	return nil, s.validateInvite(ctx, in)
}

// resetTarget reads an account that may get a password reset link: a local one (the identity provider manages
// linked ones), with owner only a subuser of owner.
func (s *Store) resetTarget(ctx context.Context, name, owner string) (*v1alpha1.User, error) {
	if name == "" {
		return nil, validation.Field("username", errors.New("name the account"))
	}
	u := &v1alpha1.User{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: name}, u)
	if err == nil && owner != "" && u.Spec.Owner != owner {
		err = apierrors.NewNotFound(schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "users"}, name)
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
	if err := s.checkShares(ctx, in.Owner, in.Shares); err != nil {
		return err
	}
	return errs.OrNil()
}

// usernameFree checks that a username is valid and not taken yet.
func (s *Store) usernameFree(ctx context.Context, name string) error {
	if err := tenancy.ValidateUsername(name); err != nil {
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

// ListInvites returns the invites, newest first, also expired ones; with an owner only the invites for its
// subusers. It reads uncached, so an invite created a moment ago is listed.
func (s *Store) ListInvites(ctx context.Context, owner string) ([]v1alpha1.Invite, error) {
	var list v1alpha1.InviteList
	if err := s.Reader.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return nil, err
	}
	if owner != "" {
		list.Items = slices.DeleteFunc(list.Items, func(i v1alpha1.Invite) bool { return i.Spec.Owner != owner })
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})
	return list.Items, nil
}

// RenewInvite gives an invite a new link, valid for the full lifetime again; the old link stops working. Expired
// invites can be renewed, too. With an owner only an invite for its subusers is found.
func (s *Store) RenewInvite(ctx context.Context, id, owner string) (*v1alpha1.Invite, string, error) {
	invite, err := s.getInvite(ctx, id, owner)
	if err != nil {
		return nil, "", err
	}
	token, hash := auth.NewInviteToken(id)
	patch := client.MergeFrom(invite.DeepCopy())
	invite.Spec.Hash = hash
	invite.Spec.ExpiresAt = metav1.NewTime(time.Now().Add(InviteLifetime))
	if err := s.Client.Patch(ctx, invite, patch); err != nil {
		return nil, "", err
	}
	return invite, token, nil
}

// DeleteInvite revokes an invite; its link stops working. With an owner only an invite for its subusers is found.
func (s *Store) DeleteInvite(ctx context.Context, id, owner string) error {
	invite, err := s.getInvite(ctx, id, owner)
	if err != nil {
		return err
	}
	return s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID})
}

// getInvite reads an invite (uncached); with an owner, invites of others are not found.
func (s *Store) getInvite(ctx context.Context, id, owner string) (*v1alpha1.Invite, error) {
	invite := &v1alpha1.Invite{}
	if err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: id}, invite); err != nil {
		return nil, err
	}
	if owner != "" && invite.Spec.Owner != owner {
		return nil, apierrors.NewNotFound(inviteResource, id)
	}
	return invite, nil
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

// AcceptInvite creates the account of an invite link and removes the invite, so the link works once.
func (s *Store) AcceptInvite(ctx context.Context, token string, in AcceptInput) (*v1alpha1.User, error) {
	// Two requests with the same link must not both create an account: the second one finds no invite.
	s.inviteUses.Lock()
	defer s.inviteUses.Unlock()
	invite, err := s.FindInvite(ctx, token)
	if err != nil {
		return nil, err
	}
	if invite.Spec.Username != "" {
		in.Username = invite.Spec.Username
	}
	if invite.Spec.Reset {
		return s.resetPassword(ctx, invite, in.Password)
	}
	owner, err := s.inviteOwner(ctx, invite)
	if err != nil {
		return nil, err
	}
	u, err := s.createFor(ctx, CreateInput{
		Username: in.Username, Password: in.Password, DisplayName: in.DisplayName, Role: invite.Spec.Role,
	}, owner)
	if err != nil {
		return nil, err
	}
	if err := s.share(ctx, invite, u.Name); err != nil {
		return nil, err
	}
	if err := s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID}); err != nil {
		return nil, err
	}
	return u, nil
}

// resetPassword sets the password of a reset link's account and removes the link. Like a reset by an administrator,
// it ends every session and API token of the account.
func (s *Store) resetPassword(ctx context.Context, invite *v1alpha1.Invite, password string) (*v1alpha1.User, error) {
	u := &v1alpha1.User{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: invite.Spec.Username}, u)
	if apierrors.IsNotFound(err) {
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
