package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
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

// setupMu serializes setup requests, so only one first administrator can be created.
var setupMu sync.Mutex

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
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	setupMu.Lock()
	defer setupMu.Unlock()
	required, err := a.needsSetup(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	if !required {
		a.fail(c, conflict(errors.New("the panel is already set up")))
		return
	}
	key := "setup|" + c.ClientIP()
	now := time.Now()
	if ok, _ := a.Limiter.Take(key, now); !ok {
		a.fail(c, tooManyRequests(errors.New("too many attempts, try again later")))
		return
	}
	if a.SetupToken == "" || !auth.EqualTokens(strings.TrimSpace(req.Token), a.SetupToken) {
		a.fail(
			c,
			fieldError(
				http.StatusForbidden, "token", errors.New("invalid setup token, use the link from the panel log"),
			),
		)
		return
	}
	u, err := a.Users.Create(c, users.CreateInput{
		Username: req.Username, Password: req.Password, DisplayName: req.DisplayName, Email: req.Email,
		Role: v1alpha1.RoleAdmin,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.Log.Info("first administrator created via the setup page", "user", u.Name)
	a.SetupToken = ""
	res, err := a.startSession(c, u)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
