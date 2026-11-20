package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/users"
	"app/base/validation"
)

// UserView is the public representation of a user (never contains hashes).
type UserView struct {
	Username    string            `json:"username"              example:"alice"`
	DisplayName string            `json:"displayName,omitempty" example:"Alice"`
	Email       string            `json:"email,omitempty"       example:"alice@example.com"`
	Role        v1alpha1.UserRole `json:"role"                  example:"admin"`
	Disabled    bool              `json:"disabled"`
	// MustChangePassword is true while the user has to replace the password set by an administrator.
	MustChangePassword bool `json:"mustChangePassword"`
	// OIDC is the user of the identity provider the account is linked to (which sets display name, email and
	// role); null for accounts that are not linked.
	OIDC *v1alpha1.OIDCIdentity `json:"oidc" extensions:"x-nullable"`
	// HasPassword is false for accounts that sign in only through the identity provider.
	HasPassword bool        `json:"hasPassword"`
	CreatedAt   time.Time   `json:"createdAt"`
	LastLoginAt *time.Time  `json:"lastLoginAt,omitempty"`
	Tokens      []TokenView `json:"tokens"`
	app.UserView
}

// TokenView describes an API token without its value.
type TokenView struct {
	ID        string    `json:"id"                  example:"1a2b3c4d5e6f"`
	Name      string    `json:"name"                example:"ci"`
	CreatedAt time.Time `json:"createdAt"`
	// ExpiresAt is the end of the token: its own expiry or the longest API token lifetime of the panel.
	ExpiresAt time.Time `json:"expiresAt"`
}

// CreateUserRequest creates a user; the role defaults to v1alpha1.DefaultRole.
type CreateUserRequest struct {
	Username    string            `json:"username"              example:"alice"`
	Password    string            `json:"password"              example:"a-long-password"`
	DisplayName string            `json:"displayName,omitempty" example:"Alice"`
	Email       string            `json:"email,omitempty"       example:"alice@example.com"`
	Role        v1alpha1.UserRole `json:"role,omitempty"        example:"admin"`
	// MustChangePassword makes the user replace the password after signing in.
	MustChangePassword bool `json:"mustChangePassword,omitempty"`
}

// Validate checks that username and password are set.
func (req CreateUserRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("username", req.Username != "")
	errs.Require("password", req.Password != "")
	return errs.OrNil()
}

// UpdateUserRequest changes a user; omitted fields stay unchanged. A new password signs the user out everywhere
// and revokes their API tokens.
type UpdateUserRequest struct {
	DisplayName *string            `json:"displayName,omitempty"`
	Email       *string            `json:"email,omitempty"`
	Role        *v1alpha1.UserRole `json:"role,omitempty"        `
	Disabled    *bool              `json:"disabled,omitempty"`
	Password    *string            `json:"password,omitempty"`
	// MustChangePassword makes the user replace the password after the next sign-in.
	MustChangePassword *bool `json:"mustChangePassword,omitempty"`
}

// changes names what an update changes, for the audit log (never the password itself).
func (r UpdateUserRequest) changes() []string {
	var out []string
	if r.DisplayName != nil {
		out = append(out, "displayName")
	}
	if r.Email != nil {
		out = append(out, "email")
	}
	if r.Role != nil {
		out = append(out, "role="+string(*r.Role))
	}
	if r.Disabled != nil {
		out = append(out, fmt.Sprintf("disabled=%t", *r.Disabled))
	}
	if r.Password != nil {
		out = append(out, "password reset")
	}
	if r.MustChangePassword != nil {
		out = append(out, fmt.Sprintf("mustChangePassword=%t", *r.MustChangePassword))
	}
	return out
}

// tokenExpiry is when a token ends: its own expiry, at the latest maxAge after its creation.
func tokenExpiry(t v1alpha1.APIToken, maxAge time.Duration) time.Time {
	end := t.CreatedAt.Add(maxAge)
	if t.ExpiresAt != nil && t.ExpiresAt.Time.Before(end) {
		return t.ExpiresAt.Time
	}
	return end
}

func tokenView(t v1alpha1.APIToken, maxAge time.Duration) TokenView {
	return TokenView{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt.Time, ExpiresAt: tokenExpiry(t, maxAge)}
}

func tokenViews(u *v1alpha1.User, maxAge time.Duration) []TokenView {
	out := []TokenView{}
	for _, t := range u.Spec.Tokens {
		out = append(out, tokenView(t, maxAge))
	}
	return out
}

// userViews describes accounts for the API, with the app's parts.
func (a *API) userViews(ctx context.Context, list []*v1alpha1.User) []UserView {
	maxAge := a.Settings.Lifetimes(ctx).APIToken
	var extra []app.UserView
	if a.App != nil {
		extra = a.App.UserViews(ctx, list)
	}
	out := make([]UserView, 0, len(list))
	for i, u := range list {
		v := UserView{
			Username: u.Name, DisplayName: u.Spec.DisplayName, Email: u.Spec.Email, Role: u.Spec.Role,
			Disabled: u.Spec.Disabled, MustChangePassword: u.Spec.MustChangePassword,
			OIDC: u.Spec.OIDC, HasPassword: u.Spec.PasswordHash != "", CreatedAt: u.CreationTimestamp.Time,
			Tokens: tokenViews(u, maxAge),
		}
		if u.Status.LastLoginAt != nil {
			v.LastLoginAt = &u.Status.LastLoginAt.Time
		}
		if extra != nil {
			v.UserView = extra[i]
		}
		out = append(out, v)
	}
	return out
}

// userView is userViews for one account.
func (a *API) userView(ctx context.Context, u *v1alpha1.User) UserView {
	return a.userViews(ctx, []*v1alpha1.User{u})[0]
}

// loadUser reads the account of the path; only accounts the caller manages are found (users.Store.Manages).
func (a *API) loadUser(w http.ResponseWriter, r *http.Request) (*v1alpha1.User, bool) {
	u := &v1alpha1.User{}
	err := a.Reader.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: r.PathValue("user")}, u)
	if err == nil && !a.Users.Manages(PrincipalOf(r).User, u) {
		err = apierrors.NewNotFound(users.UserResource, u.Name)
	}
	if err != nil {
		a.Fail(w, r, err)
		return nil, false
	}
	return u, true
}

// listUsers godoc
//
//	@Summary		List users
//	@Description	Administrators get every account, other users the accounts they own.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		UserView
//	@Failure		403	{object}	ErrorResponse
//	@Router			/users [get]
func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	var users v1alpha1.UserList
	if err := a.Client.List(r.Context(), &users, client.InNamespace(a.Namespace)); err != nil {
		a.Fail(w, r, err)
		return
	}
	sort.Slice(users.Items, func(i, j int) bool { return users.Items[i].Name < users.Items[j].Name })
	p := PrincipalOf(r)
	list := []*v1alpha1.User{}
	for i := range users.Items {
		if u := &users.Items[i]; u.DeletionTimestamp == nil && a.Users.Manages(p.User, u) {
			list = append(list, u)
		}
	}
	JSON(w, http.StatusOK, a.userViews(r.Context(), list))
}

// getUser godoc
//
//	@Summary	Get a user
//	@Tags		Users
//	@Produce	json
//	@Security	BearerAuth
//	@Param		user	path		string	true	"Username"
//	@Success	200		{object}	UserView
//	@Router		/users/{user} [get]
func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	if u, ok := a.loadUser(w, r); ok {
		JSON(w, http.StatusOK, a.userView(r.Context(), u))
	}
}

// createUser godoc
//
//	@Summary		Create a user
//	@Description	The password is stored as Argon2id hash; the app may add to the account (users.App.Created).
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateUserRequest	true	"User"
//	@Success		201		{object}	UserView
//	@Failure		409		{object}	ErrorResponse
//	@Failure		422		{object}	ErrorResponse
//	@Router			/users [post]
func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if !a.Decode(w, r, &req) {
		return
	}
	u, err := a.Users.Create(r.Context(), users.CreateInput(req))
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "user created", "user", u.Name, "role", u.Spec.Role)
	JSON(w, http.StatusCreated, a.userView(r.Context(), u))
}

// updateUser godoc
//
//	@Summary	Update a user
//	@Tags		Users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		user	path		string				true	"Username"
//	@Param		body	body		UpdateUserRequest	true	"Changes"
//	@Success	200		{object}	UserView
//	@Failure	409		{object}	ErrorResponse	"last administrator, or managed by the identity provider"
//	@Router		/users/{user} [patch]
func (a *API) updateUser(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserRequest
	if !a.Decode(w, r, &req) {
		return
	}
	u, ok := a.loadUser(w, r)
	if !ok {
		return
	}
	if err := a.Users.Update(r.Context(), u, users.UpdateInput(req)); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "user updated", "user", u.Name, "changes", req.changes())
	JSON(w, http.StatusOK, a.userView(r.Context(), u))
}

// deleteUser godoc
//
//	@Summary		Delete a user
//	@Description	Deletes the user and the accounts it owns; its sessions and API tokens stop working at once.
//	@Description	Users may delete the accounts they own.
//	@Tags			Users
//	@Security		BearerAuth
//	@Param			user	path	string	true	"Username"
//	@Success		204
//	@Failure		409	{object}	ErrorResponse
//	@Router			/users/{user} [delete]
func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	u, ok := a.loadUser(w, r)
	if !ok {
		return
	}
	if err := a.Users.Delete(r.Context(), u, PrincipalOf(r).User.Name); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "user deleted", "user", u.Name)
	w.WriteHeader(http.StatusNoContent)
}
