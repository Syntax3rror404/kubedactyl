// Package appusers adds what Kubedactyl needs to the accounts of the base (base/users): every user gets a
// namespace for game servers, and users may have subusers (accounts that belong to them) who see the servers
// shared with them.
package appusers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/settings"
	"app/base/users"
	"app/base/validation"
	"app/internal/appsettings"
	"app/internal/tenancy"
)

// Accounts adds namespaces and subusers to the accounts (users.App).
type Accounts struct {
	Client client.Client
	// Reader reads uncached: limits and invites must not come from a lagging cache.
	Reader    client.Reader
	Namespace string
	// Settings holds the subuser limit.
	Settings *settings.Store
}

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

// Created creates the namespace of a new account, so servers can be assigned right away; a subuser gets none.
func (a *Accounts) Created(ctx context.Context, u *v1alpha1.User) error {
	if Subuser(u) {
		return nil
	}
	_, err := tenancy.EnsureNamespace(ctx, a.Client, u.Name)
	return err
}

// Owner is the owner of a subuser.
func (a *Accounts) Owner(u *v1alpha1.User) string { return u.Spec.Owner }

// InviteOwner is the owner of an invite for a subuser.
func (a *Accounts) InviteOwner(i *v1alpha1.Invite) string { return i.Spec.Owner }

// PrepareInvite makes an invite of a user (always) or of an administrator who asks for it an invite for a subuser
// of by: role user, the servers of the request, at most the subuser limit of the settings. A user's reset link (for
// a subuser) belongs to the user, who manages it like the subuser.
func (a *Accounts) PrepareInvite(
	ctx context.Context, i *v1alpha1.Invite, req app.InviteRequest, by *v1alpha1.User,
) error {
	if i.Spec.Reset {
		if by.Spec.Role != v1alpha1.RoleAdmin {
			i.Spec.Owner = by.Name
		}
		return nil
	}
	if by.Spec.Role == v1alpha1.RoleAdmin && !req.Subuser {
		return nil
	}
	set, err := a.Settings.Current(ctx)
	if err != nil {
		return err
	}
	owner, err := a.subuserOwner(ctx, by.Name, appsettings.SubusersPerUser(set))
	if err != nil {
		return err
	}
	if err := a.checkShares(ctx, by.Name, req.Shares); err != nil {
		return err
	}
	i.Spec.Role, i.Spec.Owner, i.Spec.Shares = v1alpha1.RoleUser, by.Name, req.Shares
	users.OwnedBy(&i.ObjectMeta, owner)
	return nil
}

// Accept makes the account of an invite for a subuser a subuser of the invite's owner; an owner that is gone
// makes the invite invalid.
func (a *Accounts) Accept(ctx context.Context, i *v1alpha1.Invite, u *v1alpha1.User) error {
	if i.Spec.Owner == "" {
		return nil
	}
	owner := &v1alpha1.User{}
	err := a.Reader.Get(ctx, client.ObjectKey{Namespace: a.Namespace, Name: i.Spec.Owner}, owner)
	if apierrors.IsNotFound(err) || (err == nil && (Subuser(owner) || owner.DeletionTimestamp != nil)) {
		return users.ErrInvalidInvite
	}
	if err != nil {
		return err
	}
	u.Spec.Owner = owner.Name
	users.OwnedBy(&u.ObjectMeta, owner)
	return nil
}

// Accepted gives the new subuser the servers listed in its invite; servers deleted since are skipped.
func (a *Accounts) Accepted(ctx context.Context, i *v1alpha1.Invite, u *v1alpha1.User) error {
	for _, sh := range i.Spec.Shares {
		key := client.ObjectKey{Namespace: tenancy.Namespace(i.Spec.Owner), Name: sh.Server}
		share := func() error { return a.addShare(ctx, key, u.Name, sh.Access) }
		if err := retry.RetryOnConflict(retry.DefaultRetry, share); err != nil {
			return err
		}
	}
	return nil
}

// subuserOwner reads the owner of a new subuser (uncached) and checks that it may have one more: limit counts
// its subusers and invites for subusers.
func (a *Accounts) subuserOwner(ctx context.Context, name string, limit int) (*v1alpha1.User, error) {
	if limit <= 0 {
		return nil, ErrSubusersOff
	}
	owner := &v1alpha1.User{}
	if err := a.Reader.Get(ctx, client.ObjectKey{Namespace: a.Namespace, Name: name}, owner); err != nil {
		return nil, err
	}
	if Subuser(owner) {
		return nil, ErrSubuser
	}
	n, err := a.countSubusers(ctx, name)
	if err != nil {
		return nil, err
	}
	if n >= limit {
		return nil, ErrSubuserLimit
	}
	return owner, nil
}

// countSubusers counts the subusers of a user and the invites for more (uncached).
func (a *Accounts) countSubusers(ctx context.Context, owner string) (int, error) {
	var accounts v1alpha1.UserList
	if err := a.Reader.List(ctx, &accounts, client.InNamespace(a.Namespace)); err != nil {
		return 0, err
	}
	var invites v1alpha1.InviteList
	if err := a.Reader.List(ctx, &invites, client.InNamespace(a.Namespace)); err != nil {
		return 0, err
	}
	n := 0
	for _, i := range invites.Items {
		if i.Spec.Owner == owner && !i.Spec.Reset {
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

// checkShares checks the servers of an invite: each a server of the owner, once, with view or control access.
func (a *Accounts) checkShares(ctx context.Context, owner string, shares []v1alpha1.InviteShare) error {
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
		err = a.Reader.Get(ctx, key, &v1alpha1.GameServer{})
		if apierrors.IsNotFound(err) {
			return validation.Field(field, fmt.Errorf("you have no server %s", sh.Server))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// addShare gives a user access to one server. It reads the server uncached, so a retry after a conflict sees the
// latest version.
func (a *Accounts) addShare(ctx context.Context, key client.ObjectKey, user string, access v1alpha1.ShareAccess) error {
	gs := &v1alpha1.GameServer{}
	if err := a.Reader.Get(ctx, key, gs); err != nil {
		return client.IgnoreNotFound(err)
	}
	patch := client.MergeFromWithOptions(gs.DeepCopy(), client.MergeFromWithOptimisticLock{})
	gs.Spec.SharedWith = append(slices.DeleteFunc(gs.Spec.SharedWith, func(s v1alpha1.ServerShare) bool {
		return s.User == user
	}), v1alpha1.ServerShare{User: user, Access: access})
	return client.IgnoreNotFound(a.Client.Patch(ctx, gs, patch))
}
