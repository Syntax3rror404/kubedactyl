package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/validation"
)

// UpdatePasswordRequest replaces the own password; all sessions are signed out.
type UpdatePasswordRequest struct {
	Current string `json:"current" binding:"required"`
	New     string `json:"new"     binding:"required"`
	// RevokeTokens also revokes every API token of the user (after a leak).
	RevokeTokens bool `json:"revokeTokens,omitempty"`
}

// CreateTokenRequest creates a personal API token.
type CreateTokenRequest struct {
	Name string `json:"name" binding:"required" example:"ci"`
	// ExpiresInDays is the lifetime: 1 up to the longest API token lifetime of the panel.
	ExpiresInDays int `json:"expiresInDays" example:"90"`
}

// CreatedToken contains the token value, which is shown only once.
type CreatedToken struct {
	TokenView
	Token string `json:"token" example:"kdt_1a2b3c4d5e6f_…"`
}

// updatePassword godoc
//
//	@Summary		Change the own password
//	@Description	Invalidates all existing sessions of the user, with revokeTokens also the API tokens.
//	@Tags			Auth
//	@Accept			json
//	@Security		BearerAuth
//	@Param			body	body	UpdatePasswordRequest	true	"Passwords"
//	@Success		204
//	@Failure		409	{object}	ErrorResponse	"the account signs in through single sign-on only"
//	@Failure		422	{object}	ErrorResponse	"current password is wrong or the new one is too weak"
//	@Router			/auth/password [put]
func (a *API) updatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	user := principal(c).User.DeepCopy()
	if err := a.Users.UpdatePassword(c, user, req.Current, req.New, req.RevokeTokens); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "password changed", "tokensRevoked", req.RevokeTokens)
	a.logout(c)
}

// logoutAll godoc
//
//	@Summary		Sign out on all devices
//	@Description	Invalidates every session of the user (including this one). API tokens stay valid; revoke them
//	@Description	separately.
//	@Tags			Auth
//	@Security		BearerAuth
//	@Success		204
//	@Router			/auth/logout-all [post]
func (a *API) logoutAll(c *gin.Context) {
	if err := a.Users.EndSessions(c, principal(c).User.DeepCopy()); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "signed out everywhere")
	a.logout(c)
}

// listTokens godoc
//
//	@Summary	List own API tokens
//	@Tags		Auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}	TokenView
//	@Router		/auth/tokens [get]
func (a *API) listTokens(c *gin.Context) {
	c.JSON(http.StatusOK, tokenViews(principal(c).User, a.Settings.Lifetimes(c).APIToken))
}

// createToken godoc
//
//	@Summary		Create an API token
//	@Description	Use it as "Authorization: Bearer <token>". The value is returned only once. Only a session can
//	@Description	create tokens: a token cannot extend its own access.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateTokenRequest	true	"Token"
//	@Success		201		{object}	CreatedToken
//	@Router			/auth/tokens [post]
func (a *API) createToken(c *gin.Context) {
	if principal(c).TokenID != "" {
		a.fail(c, forbidden(errors.New("API tokens cannot create tokens; sign in to create one")))
		return
	}
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	lifetimes := a.Settings.Lifetimes(c)
	if req.ExpiresInDays < 1 || req.ExpiresInDays > lifetimes.APITokenDays {
		err := fmt.Errorf("must be between 1 and %d days (the longest lifetime the panel allows)",
			lifetimes.APITokenDays)
		a.fail(c, validation.Field("expiresInDays", err))
		return
	}
	user, ok := a.currentUser(c)
	if !ok {
		return
	}
	token, id, hash := auth.NewAPIToken()
	t := v1alpha1.APIToken{
		ID: id, Name: req.Name, Hash: hash, CreatedAt: metav1.Now(),
		ExpiresAt: &metav1.Time{Time: time.Now().AddDate(0, 0, req.ExpiresInDays)},
	}
	patch := client.MergeFromWithOptions(user.DeepCopy(), client.MergeFromWithOptimisticLock{})
	user.Spec.Tokens = append(user.Spec.Tokens, t)
	if err := a.Client.Patch(c, user, patch); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "API token created", "name", t.Name)
	c.JSON(http.StatusCreated, CreatedToken{TokenView: tokenView(t, lifetimes.APIToken), Token: token})
}

// deleteToken godoc
//
//	@Summary	Revoke an API token
//	@Tags		Auth
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Token id"
//	@Success	204
//	@Failure	404	{object}	ErrorResponse
//	@Router		/auth/tokens/{id} [delete]
func (a *API) deleteToken(c *gin.Context) {
	user, ok := a.currentUser(c)
	if !ok {
		return
	}
	patch := client.MergeFromWithOptions(user.DeepCopy(), client.MergeFromWithOptimisticLock{})
	kept := user.Spec.Tokens[:0]
	for _, t := range user.Spec.Tokens {
		if t.ID != c.Param("id") {
			kept = append(kept, t)
		}
	}
	if len(kept) == len(user.Spec.Tokens) {
		a.fail(c, notFound(errors.New("token not found")))
		return
	}
	user.Spec.Tokens = kept
	if err := a.Client.Patch(c, user, patch); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "API token revoked", "id", c.Param("id"))
	c.Status(http.StatusNoContent)
}

// currentUser reads the caller's user uncached: the token list is replaced as a whole, so a
// lagging cache would drop a token created a moment ago.
func (a *API) currentUser(c *gin.Context) (*v1alpha1.User, bool) {
	user := &v1alpha1.User{}
	if err := a.Reader.Get(c, client.ObjectKeyFromObject(principal(c).User), user); err != nil {
		a.fail(c, err)
		return nil, false
	}
	return user, true
}
