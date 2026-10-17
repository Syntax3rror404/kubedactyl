package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/settings"
	"app/internal/users"
)

// InviteView describes an invite without the token of its link.
type InviteView struct {
	ID string `json:"id" example:"1a2b3c4d5e6f"`
	// Username is the name the account gets; empty lets the invited person choose it.
	Username  string            `json:"username,omitempty"  example:"alice"`
	Role      v1alpha1.UserRole `json:"role"                example:"user"         enums:"admin,user"`
	Note      string            `json:"note,omitempty"      example:"Alice from the Discord server"`
	CreatedBy string            `json:"createdBy,omitempty" example:"admin"`
	// Owner is set for an invite of a subuser: the account will see the servers this user shares.
	Owner     string    `json:"owner,omitempty"     example:"bob"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	// Reset links set a new password for the existing account Username.
	Reset bool `json:"reset"`
	// Shares are the servers the subuser sees once the account exists.
	Shares []v1alpha1.InviteShare `json:"shares"`
	// Expired invites stay listed until they are revoked; their link no longer works.
	Expired bool `json:"expired"`
}

// CreatedInvite contains the token of the invite link, which is shown only once.
type CreatedInvite struct {
	InviteView
	Token string `json:"token" example:"1a2b3c4d5e6f.…"`
}

// CreateInviteRequest creates an invite link; an empty username lets the invited person choose it.
type CreateInviteRequest struct {
	Username string            `json:"username,omitempty" example:"alice"`
	Role     v1alpha1.UserRole `json:"role,omitempty"     example:"user"                          enums:"admin,user"`
	Note     string            `json:"note,omitempty"     example:"Alice from the Discord server"`
	// Subuser invites a subuser of the caller (role user). Users always invite subusers.
	Subuser bool `json:"subuser,omitempty"`
	// Shares are servers of the caller the subuser sees once the account exists.
	Shares []v1alpha1.InviteShare `json:"shares,omitempty"`
	// Reset creates a link that sets a new password for the existing account Username (users: their subusers).
	Reset bool `json:"reset,omitempty"`
}

// InviteDetails is what the invite page needs before the account is created.
type InviteDetails struct {
	// Username is set when the name was chosen for the invited person; they cannot change it.
	Username  string    `json:"username,omitempty" example:"alice"`
	ExpiresAt time.Time `json:"expiresAt"`
	// Reset links set a new password for the account Username instead of creating one.
	Reset bool `json:"reset"`
}

// AcceptInviteRequest creates the account of an invite link.
type AcceptInviteRequest struct {
	Token string `json:"token" binding:"required"`
	// Username is ignored when the invite names one.
	Username    string `json:"username,omitempty"    example:"alice"`
	DisplayName string `json:"displayName,omitempty" example:"Alice"`
	Password    string `json:"password"              binding:"required" example:"a-long-password"`
}

func inviteView(i *v1alpha1.Invite) InviteView {
	return InviteView{
		ID: i.Name, Username: i.Spec.Username, Role: i.Spec.Role, Note: i.Spec.Note, CreatedBy: i.Spec.CreatedBy,
		Owner: i.Spec.Owner, Reset: i.Spec.Reset, Shares: append([]v1alpha1.InviteShare{}, i.Spec.Shares...),
		CreatedAt: i.CreationTimestamp.Time, ExpiresAt: i.Spec.ExpiresAt.Time,
		Expired: !time.Now().Before(i.Spec.ExpiresAt.Time),
	}
}

// listInvites godoc
//
//	@Summary		List invites
//	@Description	Open and expired invites, newest first: administrators get all, users those of their subusers.
//	@Description	The links themselves are not stored.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}	InviteView
//	@Router			/invites [get]
func (a *API) listInvites(c *gin.Context) {
	list, err := a.Users.ListInvites(c, inviteScope(principal(c)))
	if err != nil {
		a.fail(c, err)
		return
	}
	out := []InviteView{}
	for i := range list {
		out = append(out, inviteView(&list[i]))
	}
	c.JSON(http.StatusOK, out)
}

// createInvite godoc
//
//	@Summary		Create an invite link
//	@Description	The link (/invite?token=<token>) lets someone create an account once within 7 days, or with
//	@Description	reset set a new password for an existing one. The token is returned only once. Invites of
//	@Description	subusers count against the subuser limit of the settings.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateInviteRequest	true	"Invite"
//	@Success		201		{object}	CreatedInvite
//	@Failure		422		{object}	ErrorResponse
//	@Router			/invites [post]
func (a *API) createInvite(c *gin.Context) {
	var req CreateInviteRequest
	if !a.bindJSON(c, &req) {
		return
	}
	p := principal(c)
	in := users.InviteInput{Username: req.Username, Role: req.Role, Note: req.Note, Reset: req.Reset}
	switch {
	case req.Reset:
		// Users reset only their subusers.
		if !p.Admin() {
			in.Owner = p.User.Name
		}
	case req.Subuser || !p.Admin():
		set, err := a.Settings.Current(c)
		if err != nil {
			a.fail(c, err)
			return
		}
		in.Owner, in.Limit, in.Shares = p.User.Name, settings.SubusersPerUser(set), req.Shares
	}
	invite, token, err := a.Users.CreateInvite(c, in, p.User.Name)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "invite created", "invite", invite.Name, "username", invite.Spec.Username, "role", invite.Spec.Role,
		"owner", invite.Spec.Owner, "reset", invite.Spec.Reset)
	c.JSON(http.StatusCreated, CreatedInvite{InviteView: inviteView(invite), Token: token})
}

// renewInvite godoc
//
//	@Summary		Renew an invite link
//	@Description	Replaces the link of an invite (the old one stops working) and makes it valid for 7 days again,
//	@Description	also after it expired. The new token is returned only once.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			invite	path		string	true	"Invite id"
//	@Success		200		{object}	CreatedInvite
//	@Failure		404		{object}	ErrorResponse
//	@Router			/invites/{invite}/renew [post]
func (a *API) renewInvite(c *gin.Context) {
	invite, token, err := a.Users.RenewInvite(c, c.Param("invite"), inviteScope(principal(c)))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "invite renewed", "invite", invite.Name)
	c.JSON(http.StatusOK, CreatedInvite{InviteView: inviteView(invite), Token: token})
}

// deleteInvite godoc
//
//	@Summary	Revoke an invite
//	@Tags		Users
//	@Security	BearerAuth
//	@Param		invite	path	string	true	"Invite id"
//	@Success	204
//	@Failure	404	{object}	ErrorResponse
//	@Router		/invites/{invite} [delete]
func (a *API) deleteInvite(c *gin.Context) {
	if err := a.Users.DeleteInvite(c, c.Param("invite"), inviteScope(principal(c))); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "invite revoked", "invite", c.Param("invite"))
	c.Status(http.StatusNoContent)
}

// inviteScope is the owner whose invites the caller manages; "" (all) for administrators.
func inviteScope(p *Principal) string {
	if p.Admin() {
		return ""
	}
	return p.User.Name
}

// findInvite returns the invite of a link. Failures count against the client like failed sign-ins, so nobody
// can make the panel look up invites without end.
func (a *API) findInvite(c *gin.Context, token string) (*v1alpha1.Invite, bool) {
	key := "invite|" + c.ClientIP()
	now := time.Now()
	if ok, _ := a.Limiter.Take(key, now); !ok {
		a.fail(c, tooManyRequests(errTooManyAttempts))
		return nil, false
	}
	invite, err := a.Users.FindInvite(c, strings.TrimSpace(token))
	if !errors.Is(err, users.ErrInvalidInvite) {
		a.Limiter.Forgive(key, now)
	}
	if err != nil {
		a.fail(c, err)
		return nil, false
	}
	return invite, true
}

// getInvite godoc
//
//	@Summary		Check an invite link
//	@Description	Public: tells the invite page whether the link works and whether the username is fixed.
//	@Tags			Auth
//	@Produce		json
//	@Param			token	query		string	true	"Token of the invite link"
//	@Success		200		{object}	InviteDetails
//	@Failure		404		{object}	ErrorResponse	"unknown, used or expired link"
//	@Failure		429		{object}	ErrorResponse
//	@Router			/auth/invite [get]
func (a *API) getInvite(c *gin.Context) {
	if invite, ok := a.findInvite(c, c.Query("token")); ok {
		c.JSON(http.StatusOK, InviteDetails{
			Username: invite.Spec.Username, ExpiresAt: invite.Spec.ExpiresAt.Time, Reset: invite.Spec.Reset,
		})
	}
}

// acceptInvite godoc
//
//	@Summary		Accept an invite
//	@Description	Public: creates the account of an invite link, or sets the new password of a reset link (the
//	@Description	link works once), and signs it in.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		AcceptInviteRequest	true	"Account"
//	@Success		201		{object}	LoginResponse
//	@Failure		404		{object}	ErrorResponse	"unknown, used or expired link"
//	@Failure		422		{object}	ErrorResponse
//	@Failure		429		{object}	ErrorResponse
//	@Router			/auth/invite [post]
func (a *API) acceptInvite(c *gin.Context) {
	var req AcceptInviteRequest
	if !a.bindJSON(c, &req) {
		return
	}
	invite, ok := a.findInvite(c, req.Token)
	if !ok {
		return
	}
	u, err := a.Users.AcceptInvite(c, strings.TrimSpace(req.Token), users.AcceptInput{
		Username: req.Username, DisplayName: req.DisplayName, Password: req.Password,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	msg := "account created through an invite"
	if invite.Spec.Reset {
		msg = "password set through a reset link"
	}
	a.Log.Info(msg, "user", u.Name, "invite", invite.Name, "invitedBy", invite.Spec.CreatedBy)
	a.sendSession(c, http.StatusCreated, u)
}
