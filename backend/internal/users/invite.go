package users

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/tenancy"
	"app/internal/validation"
)

// InviteLifetime is how long an invite link works.
const InviteLifetime = 7 * 24 * time.Hour

// ErrInvalidInvite is returned for an invite link that is unknown, used or expired.
var ErrInvalidInvite = errors.New("the invite link is invalid or has expired, ask your administrator for a new one")

// InviteInput is an invite to create; an empty username lets the invited person choose it.
type InviteInput struct {
	Username string
	Role     v1alpha1.UserRole
	Note     string
}

// AcceptInput is the account an invited person creates; Username is ignored when the invite names one.
type AcceptInput struct {
	Username, DisplayName, Password string
}

// CreateInvite stores a new invite and returns it together with the token of its link, which is not stored.
func (s *Store) CreateInvite(ctx context.Context, in InviteInput, by string) (*v1alpha1.Invite, string, error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	if in.Role == "" {
		in.Role = v1alpha1.RoleUser
	}
	if err := s.validateInvite(ctx, in); err != nil {
		return nil, "", err
	}
	id := auth.NewInviteID()
	token, hash := auth.NewInviteToken(id)
	invite := &v1alpha1.Invite{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: s.Namespace},
		Spec: v1alpha1.InviteSpec{
			Username: in.Username, Role: in.Role, Note: strings.TrimSpace(in.Note), Hash: hash,
			ExpiresAt: metav1.NewTime(time.Now().Add(InviteLifetime)), CreatedBy: by,
		},
	}
	if err := s.Client.Create(ctx, invite); err != nil {
		return nil, "", err
	}
	return invite, token, nil
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

// ListInvites returns the invites, newest first, also expired ones. It reads uncached, so an invite created a
// moment ago is listed.
func (s *Store) ListInvites(ctx context.Context) ([]v1alpha1.Invite, error) {
	var list v1alpha1.InviteList
	if err := s.Reader.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return nil, err
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})
	return list.Items, nil
}

// RenewInvite gives an invite a new link, valid for the full lifetime again; the old link stops working. Expired
// invites can be renewed, too.
func (s *Store) RenewInvite(ctx context.Context, id string) (*v1alpha1.Invite, string, error) {
	invite := &v1alpha1.Invite{}
	if err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: id}, invite); err != nil {
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

// DeleteInvite revokes an invite; its link stops working.
func (s *Store) DeleteInvite(ctx context.Context, id string) error {
	return s.Client.Delete(ctx, &v1alpha1.Invite{ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: s.Namespace}})
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
	u, err := s.Create(ctx, CreateInput{
		Username: in.Username, Password: in.Password, DisplayName: in.DisplayName, Role: invite.Spec.Role,
	})
	if err != nil {
		return nil, err
	}
	if err := s.Client.Delete(ctx, invite, client.Preconditions{UID: &invite.UID}); err != nil {
		return nil, err
	}
	return u, nil
}
