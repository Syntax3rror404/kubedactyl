package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/tenancy"
	"app/internal/users"
)

// UserView is the public representation of a user (never contains hashes).
type UserView struct {
	Username    string            `json:"username"              example:"alice"`
	DisplayName string            `json:"displayName,omitempty" example:"Alice"`
	Email       string            `json:"email,omitempty"       example:"alice@example.com"`
	Role        v1alpha1.UserRole `json:"role"                  example:"user"                  enums:"admin,user"`
	Disabled    bool              `json:"disabled"`
	// MustChangePassword: the user has to replace the password set by an administrator first.
	MustChangePassword bool `json:"mustChangePassword"`
	// OIDC is the user of the identity provider the account is linked to (which sets display name, email and
	// role); null for accounts that are not linked.
	OIDC *v1alpha1.OIDCIdentity `json:"oidc" extensions:"x-nullable"`
	// HasPassword is false for accounts that sign in only through the identity provider.
	HasPassword bool        `json:"hasPassword"`
	Namespace   string      `json:"namespace"             example:"kubedactyl-user-alice"`
	CreatedAt   time.Time   `json:"createdAt"`
	LastLoginAt *time.Time  `json:"lastLoginAt,omitempty"`
	Servers     int         `json:"servers"               example:"2"`
	Tokens      []TokenView `json:"tokens"`
}

// TokenView describes an API token without its value.
type TokenView struct {
	ID        string    `json:"id"                  example:"1a2b3c4d5e6f"`
	Name      string    `json:"name"                example:"ci"`
	CreatedAt time.Time `json:"createdAt"`
	// ExpiresAt is the end of the token: its own expiry or the longest API token lifetime of the panel.
	ExpiresAt time.Time `json:"expiresAt"`
}

// CreateUserRequest creates a user and its namespace.
type CreateUserRequest struct {
	Username    string            `json:"username"              binding:"required" example:"alice"`
	Password    string            `json:"password"              binding:"required" example:"a-long-password"`
	DisplayName string            `json:"displayName,omitempty"                    example:"Alice"`
	Email       string            `json:"email,omitempty"                          example:"alice@example.com"`
	Role        v1alpha1.UserRole `json:"role,omitempty"                           example:"user"              enums:"admin,user"`
	// MustChangePassword makes the user replace the password after signing in.
	MustChangePassword bool `json:"mustChangePassword,omitempty"`
}

// UpdateUserRequest changes a user; omitted fields stay unchanged. A new password signs the user out everywhere
// and revokes their API tokens.
type UpdateUserRequest struct {
	DisplayName *string            `json:"displayName,omitempty"`
	Email       *string            `json:"email,omitempty"`
	Role        *v1alpha1.UserRole `json:"role,omitempty"        enums:"admin,user"`
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

func (a *API) userView(c *gin.Context, u *v1alpha1.User) UserView {
	v := UserView{
		Username: u.Name, DisplayName: u.Spec.DisplayName, Email: u.Spec.Email, Role: u.Spec.Role,
		Disabled: u.Spec.Disabled, MustChangePassword: u.Spec.MustChangePassword,
		OIDC: u.Spec.OIDC, HasPassword: u.Spec.PasswordHash != "",
		Namespace: tenancy.Namespace(u.Name), CreatedAt: u.CreationTimestamp.Time,
		Tokens: tokenViews(u, a.Settings.Lifetimes(c).APIToken),
	}
	if u.Status.LastLoginAt != nil {
		v.LastLoginAt = &u.Status.LastLoginAt.Time
	}
	var servers v1alpha1.GameServerList
	if err := a.Client.List(c, &servers, client.InNamespace(v.Namespace)); err == nil {
		v.Servers = len(servers.Items)
	}
	return v
}

func (a *API) loadUser(c *gin.Context) (*v1alpha1.User, bool) {
	u := &v1alpha1.User{}
	if err := a.Reader.Get(c, client.ObjectKey{Namespace: a.Opts.Namespace, Name: c.Param("user")}, u); err != nil {
		a.fail(c, err)
		return nil, false
	}
	return u, true
}

// listUsers godoc
//
//	@Summary	List users
//	@Tags		Users
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}		UserView
//	@Failure	403	{object}	ErrorResponse
//	@Router		/users [get]
func (a *API) listUsers(c *gin.Context) {
	var users v1alpha1.UserList
	if err := a.Client.List(c, &users, client.InNamespace(a.Opts.Namespace)); err != nil {
		a.fail(c, err)
		return
	}
	sort.Slice(users.Items, func(i, j int) bool { return users.Items[i].Name < users.Items[j].Name })
	out := []UserView{}
	for i := range users.Items {
		if users.Items[i].DeletionTimestamp == nil {
			out = append(out, a.userView(c, &users.Items[i]))
		}
	}
	c.JSON(http.StatusOK, out)
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
func (a *API) getUser(c *gin.Context) {
	if u, ok := a.loadUser(c); ok {
		c.JSON(http.StatusOK, a.userView(c, u))
	}
}

// createUser godoc
//
//	@Summary		Create a user
//	@Description	Creates the user and its namespace (kubedactyl-user-<username>). The password is stored as
//	@Description	Argon2id hash.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateUserRequest	true	"User"
//	@Success		201		{object}	UserView
//	@Failure		409		{object}	ErrorResponse
//	@Failure		422		{object}	ErrorResponse
//	@Router			/users [post]
func (a *API) createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	u, err := a.Users.Create(c, users.CreateInput(req))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "user created", "user", u.Name, "role", u.Spec.Role)
	c.JSON(http.StatusCreated, a.userView(c, u))
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
func (a *API) updateUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	u, ok := a.loadUser(c)
	if !ok {
		return
	}
	if err := a.Users.Update(c, u, users.UpdateInput(req)); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "user updated", "user", u.Name, "changes", req.changes())
	c.JSON(http.StatusOK, a.userView(c, u))
}

// deleteUser godoc
//
//	@Summary		Delete a user
//	@Description	Deletes the user, all its game servers, their data and the user namespace.
//	@Tags			Users
//	@Security		BearerAuth
//	@Param			user	path	string	true	"Username"
//	@Success		204
//	@Failure		409	{object}	ErrorResponse
//	@Router			/users/{user} [delete]
func (a *API) deleteUser(c *gin.Context) {
	u, ok := a.loadUser(c)
	if !ok {
		return
	}
	if err := a.Users.Delete(c, u, principal(c).User.Name); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "user deleted", "user", u.Name)
	c.Status(http.StatusNoContent)
}
