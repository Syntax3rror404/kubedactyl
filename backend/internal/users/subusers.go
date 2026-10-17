package users

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/tenancy"
	"app/internal/validation"
)

// Errors of subusers.
var (
	ErrSubusersOff  = errors.New("subusers are turned off")
	ErrSubuserLimit = errors.New("you have reached the most subusers you may have, remove one first")
	// ErrSubuser means a subuser can neither own servers nor be an administrator.
	ErrSubuser = errors.New("a subuser can neither own servers nor be an administrator")
)

// Subuser reports whether the account belongs to another user (UserSpec.Owner).
func Subuser(u *v1alpha1.User) bool { return u.Spec.Owner != "" }

// Home is the namespace of the servers an account sees: its own, for a subuser its owner's.
func Home(u *v1alpha1.User) string { return tenancy.Namespace(cmp.Or(u.Spec.Owner, u.Name)) }

// CheckAccess fills in view for an empty access and refuses anything but view and control.
func CheckAccess(a v1alpha1.ShareAccess) (v1alpha1.ShareAccess, error) {
	a = cmp.Or(a, v1alpha1.ShareView)
	if a != v1alpha1.ShareView && a != v1alpha1.ShareControl {
		return a, errors.New("access must be view or control")
	}
	return a, nil
}

// ownedBy makes Kubernetes delete obj together with the owner's account.
func ownedBy(obj *metav1.ObjectMeta, owner *v1alpha1.User) {
	obj.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: v1alpha1.GroupVersion.String(), Kind: "User", Name: owner.Name, UID: owner.UID,
	}}
}

// subuserOwner reads the owner of a new subuser (uncached) and checks that it may have one more: limit counts
// its subusers and invites for subusers.
func (s *Store) subuserOwner(ctx context.Context, name string, limit int) (*v1alpha1.User, error) {
	if limit <= 0 {
		return nil, ErrSubusersOff
	}
	owner := &v1alpha1.User{}
	if err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: name}, owner); err != nil {
		return nil, err
	}
	if Subuser(owner) {
		return nil, ErrSubuser
	}
	n, err := s.countSubusers(ctx, name)
	if err != nil {
		return nil, err
	}
	if n >= limit {
		return nil, ErrSubuserLimit
	}
	return owner, nil
}

// countSubusers counts the subusers of a user and the invites for more (uncached).
func (s *Store) countSubusers(ctx context.Context, owner string) (int, error) {
	var accounts v1alpha1.UserList
	if err := s.Reader.List(ctx, &accounts, client.InNamespace(s.Namespace)); err != nil {
		return 0, err
	}
	invites, err := s.ListInvites(ctx, owner)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, i := range invites {
		if !i.Spec.Reset {
			n++
		}
	}
	for i := range accounts.Items {
		if accounts.Items[i].Spec.Owner == owner {
			n++
		}
	}
	return n, nil
}

// inviteOwner returns the owner of an invite for a subuser, nil for other invites. An owner that is gone makes
// the invite invalid.
func (s *Store) inviteOwner(ctx context.Context, invite *v1alpha1.Invite) (*v1alpha1.User, error) {
	if invite.Spec.Owner == "" {
		return nil, nil
	}
	owner := &v1alpha1.User{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: invite.Spec.Owner}, owner)
	if apierrors.IsNotFound(err) || (err == nil && (Subuser(owner) || owner.DeletionTimestamp != nil)) {
		return nil, ErrInvalidInvite
	}
	return owner, err
}

// checkShares checks the servers of an invite: each a server of the owner, once, with view or control access.
func (s *Store) checkShares(ctx context.Context, owner string, shares []v1alpha1.InviteShare) error {
	seen := map[string]bool{}
	for i := range shares {
		sh := &shares[i]
		field := fmt.Sprintf("shares.%d", i)
		var err error
		if sh.Access, err = CheckAccess(sh.Access); err != nil {
			return validation.Field(field, err)
		}
		if seen[sh.Server] {
			return validation.Field(field, fmt.Errorf("%s is listed twice", sh.Server))
		}
		seen[sh.Server] = true
		key := client.ObjectKey{Namespace: tenancy.Namespace(owner), Name: sh.Server}
		err = s.Reader.Get(ctx, key, &v1alpha1.GameServer{})
		if apierrors.IsNotFound(err) {
			return validation.Field(field, fmt.Errorf("you have no server %s", sh.Server))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// share gives the new account of an invite the servers listed in it; servers deleted since are skipped.
func (s *Store) share(ctx context.Context, invite *v1alpha1.Invite, user string) error {
	for _, sh := range invite.Spec.Shares {
		gs := &v1alpha1.GameServer{}
		key := client.ObjectKey{Namespace: tenancy.Namespace(invite.Spec.Owner), Name: sh.Server}
		if err := s.Reader.Get(ctx, key, gs); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		patch := client.MergeFromWithOptions(gs.DeepCopy(), client.MergeFromWithOptimisticLock{})
		gs.Spec.SharedWith = append(slices.DeleteFunc(gs.Spec.SharedWith, func(s v1alpha1.ServerShare) bool {
			return s.User == user
		}), v1alpha1.ServerShare{User: user, Access: sh.Access})
		if err := s.Client.Patch(ctx, gs, patch); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}
