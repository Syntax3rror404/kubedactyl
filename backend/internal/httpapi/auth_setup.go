package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/users"
)

// SetupStatus tells the web UI whether the panel still needs its first administrator.
type SetupStatus struct {
	Required bool `json:"required"`
}

// SetupRequest creates the first administrator.
type SetupRequest struct {
	Username    string `json:"username"              binding:"required" example:"admin"`
	Password    string `json:"password"              binding:"required" example:"a-long-password"`
	DisplayName string `json:"displayName,omitempty"                    example:"Administrator"`
	Email       string `json:"email,omitempty"`
	// Token is the one-time setup token from the panel log (or KUBEDACTYL_SETUP_TOKEN).
	Token string `json:"token" binding:"required"`
}

// needsSetup reports whether no active administrator exists. The setup routes are public: once the cache
// knows an administrator they need no call to the API server; until then the answer is read uncached, so it
// changes right after the setup.
func (a *API) needsSetup(ctx context.Context) (bool, error) {
	if a.Users.AdminCached(ctx) {
		return false, nil
	}
	n, err := a.Users.ActiveAdmins(ctx)
	return n == 0, err
}

// getSetupStatus godoc
//
//	@Summary		Setup state
//	@Description	"required" is true while no active administrator exists; the web UI then shows the setup page.
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	SetupStatus
//	@Router			/setup [get]
func (a *API) getSetupStatus(c *gin.Context) {
	required, err := a.needsSetup(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, SetupStatus{Required: required})
}

// runSetup godoc
//
//	@Summary		Create the first administrator
//	@Description	Only possible while no active administrator exists and with the setup token from the panel log.
//	@Description	Signs the new administrator in.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SetupRequest	true	"Administrator"
//	@Success		201		{object}	LoginResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		429		{object}	ErrorResponse
//	@Failure		422		{object}	ErrorResponse
//	@Router			/setup [post]
func (a *API) runSetup(c *gin.Context) {
	var req SetupRequest
	if !a.bindJSON(c, &req) {
		return
	}
	// Counted before the uncached admin count, so anonymous requests cannot repeat that read without end.
	now := time.Now()
	key, err := a.takeClientAttempt(c, "setup", now)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.setupMu.Lock()
	defer a.setupMu.Unlock()
	u, err := a.createFirstAdmin(c, req)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.Limiter.Forgive(key, now)
	a.Log.Info("first administrator created via the setup page", "user", u.Name)
	a.SetupToken = ""
	a.sendSession(c, http.StatusCreated, u)
}

// createFirstAdmin creates the administrator of a setup request while no active one exists (guarded by setupMu).
func (a *API) createFirstAdmin(c *gin.Context, req SetupRequest) (*v1alpha1.User, error) {
	required, err := a.needsSetup(c)
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, conflict(errors.New("the panel is already set up"))
	}
	if a.SetupToken == "" || !auth.EqualTokens(strings.TrimSpace(req.Token), a.SetupToken) {
		err := errors.New("invalid setup token, use the link from the panel log")
		return nil, fieldError(http.StatusForbidden, "token", err)
	}
	return a.Users.Create(c, users.CreateInput{
		Username: req.Username, Password: req.Password, DisplayName: req.DisplayName, Email: req.Email,
		Role: v1alpha1.RoleAdmin,
	})
}
