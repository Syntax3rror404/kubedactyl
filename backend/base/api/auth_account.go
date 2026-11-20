package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/base/auth"
	"app/base/validation"
)

// UpdatePasswordRequest replaces the own password; all sessions are signed out.
type UpdatePasswordRequest struct {
	Current string `json:"current" maxLength:"1024"`
	New     string `json:"new"     maxLength:"1024"`
	// RevokeTokens also revokes every API token of the user (after a leak).
	RevokeTokens bool `json:"revokeTokens,omitempty"`
}

// Validate checks that both passwords are set and not too long.
func (req UpdatePasswordRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("current", req.Current != "")
	errs.MaxLength("current", req.Current, 1024)
	errs.Require("new", req.New != "")
	errs.MaxLength("new", req.New, 1024)
	return errs.OrNil()
}

// CreateTokenRequest creates a personal API token.
type CreateTokenRequest struct {
	Name string `json:"name" maxLength:"64" example:"ci"`
	// ExpiresInDays is the lifetime of the token (days), from 1 to the longest API token lifetime of the panel.
	ExpiresInDays int `json:"expiresInDays" example:"90"`
}

// Validate checks that the token has a name of at most 64 characters.
func (req CreateTokenRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("name", req.Name != "")
	errs.MaxLength("name", req.Name, 64)
	return errs.OrNil()
}

// CreatedToken contains the token value, which is shown only once.
type CreatedToken struct {
	TokenView
	Token string `json:"token" example:"xyz_1a2b3c4d5e6f_…"`
}

// getMe godoc
//
//	@Summary	Current user
//	@Tags		Auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	UserView
//	@Failure	401	{object}	ErrorResponse
//	@Router		/auth/me [get]
func (a *API) getMe(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, a.userView(r.Context(), PrincipalOf(r).User))
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
func (a *API) updatePassword(w http.ResponseWriter, r *http.Request) {
	var req UpdatePasswordRequest
	if !a.Decode(w, r, &req) {
		return
	}
	user, ok := a.currentUser(w, r)
	if !ok {
		return
	}
	// Counted like a sign-in, so a stolen session cannot guess the current password without end.
	now := time.Now()
	key := "password|" + user.Name
	if ok, wait := a.Limiter.Take(key, now); !ok {
		retryAfter(w, wait)
		a.Fail(w, r, TooManyRequests(errTooManyAttempts))
		return
	}
	if err := a.Users.UpdatePassword(r.Context(), user, req.Current, req.New, req.RevokeTokens); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Limiter.Forgive(key, now)
	a.Audit(r, "password changed", "tokensRevoked", req.RevokeTokens)
	a.logout(w, r)
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
func (a *API) logoutAll(w http.ResponseWriter, r *http.Request) {
	user, ok := a.currentUser(w, r)
	if !ok {
		return
	}
	if err := a.Users.EndSessions(r.Context(), user); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "signed out everywhere")
	a.logout(w, r)
}

// listTokens godoc
//
//	@Summary	List own API tokens
//	@Tags		Auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}	TokenView
//	@Router		/auth/tokens [get]
func (a *API) listTokens(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, tokenViews(PrincipalOf(r).User, a.Settings.Lifetimes(r.Context()).APIToken))
}

// createToken godoc
//
//	@Summary		Create an API token
//	@Description	Use it as "Authorization: Bearer <token>". The value is returned only once. Only a session can
//	@Description	create tokens: a token cannot extend its own access. An account has at most 50 tokens.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateTokenRequest	true	"Token"
//	@Success		201		{object}	CreatedToken
//	@Failure		409		{object}	ErrorResponse	"the account has the most tokens it may have"
//	@Router			/auth/tokens [post]
func (a *API) createToken(w http.ResponseWriter, r *http.Request) {
	if PrincipalOf(r).TokenID != "" {
		a.Fail(w, r, Forbidden(errors.New("API tokens cannot create tokens; sign in to create one")))
		return
	}
	var req CreateTokenRequest
	if !a.Decode(w, r, &req) {
		return
	}
	lifetimes := a.Settings.Lifetimes(r.Context())
	if req.ExpiresInDays < 1 || req.ExpiresInDays > lifetimes.APITokenDays {
		err := fmt.Errorf("must be between 1 and %d days (the longest lifetime the panel allows)",
			lifetimes.APITokenDays)
		a.Fail(w, r, validation.Field("expiresInDays", err))
		return
	}
	user, ok := a.currentUser(w, r)
	if !ok {
		return
	}
	token, id, hash := auth.NewAPIToken()
	t := v1alpha1.APIToken{
		ID: id, Name: req.Name, Hash: hash, CreatedAt: metav1.Now(),
		ExpiresAt: &metav1.Time{Time: time.Now().AddDate(0, 0, req.ExpiresInDays)},
	}
	if err := a.Users.AddToken(r.Context(), user, t); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "API token created", "name", t.Name)
	JSON(w, http.StatusCreated, CreatedToken{TokenView: tokenView(t, lifetimes.APIToken), Token: token})
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
func (a *API) deleteToken(w http.ResponseWriter, r *http.Request) {
	user, ok := a.currentUser(w, r)
	if !ok {
		return
	}
	patch := client.MergeFromWithOptions(user.DeepCopy(), client.MergeFromWithOptimisticLock{})
	kept := user.Spec.Tokens[:0]
	for _, t := range user.Spec.Tokens {
		if t.ID != r.PathValue("id") {
			kept = append(kept, t)
		}
	}
	if len(kept) == len(user.Spec.Tokens) {
		a.Fail(w, r, NotFound(errors.New("token not found")))
		return
	}
	user.Spec.Tokens = kept
	if err := a.Client.Patch(r.Context(), user, patch); err != nil {
		a.Fail(w, r, err)
		return
	}
	a.Audit(r, "API token revoked", "id", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

// currentUser reads the caller's user uncached: the token list is replaced as a whole, so a
// lagging cache would drop a token created a moment ago.
func (a *API) currentUser(w http.ResponseWriter, r *http.Request) (*v1alpha1.User, bool) {
	user := &v1alpha1.User{}
	if err := a.Reader.Get(r.Context(), client.ObjectKeyFromObject(PrincipalOf(r).User), user); err != nil {
		a.Fail(w, r, err)
		return nil, false
	}
	return user, true
}
