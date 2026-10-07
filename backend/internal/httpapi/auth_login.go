package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
)

// loginLimit is a limiter of failed sign-ins and the key a sign-in counts under.
type loginLimit struct {
	limiter *auth.Limiter
	key     string
}

// DeviceCookie marks a browser in which the user signed in before (path deviceCookiePath only): its sign-ins
// do not count against the account, so failed attempts from elsewhere cannot lock the user out.
const (
	DeviceCookie     = "kd_device"
	deviceCookiePath = "/api/auth/login"
	deviceLifetime   = 180 * 24 * time.Hour
)

// loginLimits are the limits of a sign-in: client and user (reset by a success), the client
// alone (password spraying) and the user alone (from many addresses or a spoofed X-Forwarded-For),
// except from a browser the user signed in with before.
func (a *API) loginLimits(c *gin.Context, clientIP, username string, now time.Time) []loginLimit {
	limits := []loginLimit{{a.Limiter, clientIP + "|" + username}, {a.ClientLimiter, clientIP}}
	if device, err := c.Cookie(DeviceCookie); err != nil || !a.Signer.KnownDevice(device, username, now) {
		limits = append(limits, loginLimit{a.AccountLimiter, username})
	}
	return limits
}

// takeAttempt counts a sign-in under every limit before the password is checked, so parallel attempts
// cannot pass before the first one failed; when one limit refuses, the others are taken back.
func takeAttempt(limits []loginLimit, now time.Time) (time.Duration, bool) {
	for i, l := range limits {
		if ok, wait := l.limiter.Take(l.key, now); !ok {
			forgiveAttempt(limits[:i], now)
			return wait, false
		}
	}
	return 0, true
}

func forgiveAttempt(limits []loginLimit, now time.Time) {
	for _, l := range limits {
		l.limiter.Forgive(l.key, now)
	}
}

// LoginRequest authenticates with username and password.
type LoginRequest struct {
	// The maximum lengths (usernames have at most 32 characters, auth.MaxPasswordLength) keep a request from
	// filling the limiters and the log with huge keys.
	Username string `json:"username" binding:"required,max=64"   example:"admin"`
	Password string `json:"password" binding:"required,max=1024" example:"secret-password"`
}

// LoginResponse returns the session token (also set as HttpOnly cookie).
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      UserView  `json:"user"`
}

// login godoc
//
//	@Summary		Log in
//	@Description	Returns a session token and sets it as HttpOnly cookie. Repeated failures are throttled.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Credentials"
//	@Success		200		{object}	LoginResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		429		{object}	ErrorResponse
//	@Router			/auth/login [post]
func (a *API) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	// ClientIP honors X-Forwarded-For only from trusted proxies (--trusted-proxies).
	clientIP := c.ClientIP()
	now := time.Now()
	limits := a.loginLimits(c, clientIP, username, now)
	if wait, ok := takeAttempt(limits, now); !ok {
		c.Header("Retry-After", strings.TrimSuffix(wait.Round(time.Second).String(), "s"))
		a.fail(c, tooManyRequests(errors.New("too many failed logins, try again later")))
		return
	}
	user, ok := a.checkPassword(c, username, req.Password)
	if !ok {
		a.Log.Warn("sign-in failed", "user", username, "ip", clientIP)
		a.fail(c, unauthorized(errors.New("invalid username or password")))
		return
	}
	a.Limiter.Reset(limits[0].key)
	forgiveAttempt(limits[1:], now)
	a.Log.Info("signed in", "user", user.Name, "ip", clientIP)
	if err := a.Users.UpdatePasswordHash(c, user, req.Password); err != nil {
		a.Log.Warn("updating the password hash", "user", user.Name, "err", err)
	}
	res, err := a.startSession(c, user)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// checkPassword returns the user when the password matches and the account is enabled. The password is
// checked for unknown, disabled and single sign-on only users, too: the answer takes as long either way.
func (a *API) checkPassword(ctx context.Context, username, password string) (*v1alpha1.User, bool) {
	user := &v1alpha1.User{}
	err := a.Client.Get(ctx, client.ObjectKey{Namespace: a.Opts.Namespace, Name: username}, user)
	if err != nil || user.Spec.PasswordHash == "" {
		auth.VerifyDummy(password)
		return nil, false
	}
	return user, auth.VerifyPassword(password, user.Spec.PasswordHash) && !user.Spec.Disabled
}

// startSession signs a session token for the user and sets it as cookie.
func (a *API) startSession(c *gin.Context, user *v1alpha1.User) (LoginResponse, error) {
	now := time.Now()
	expires := now.Add(a.Settings.Lifetimes(c).Session)
	token, err := a.Signer.Sign(auth.Session{
		User: user.Name, Epoch: user.Spec.SessionEpoch, IssuedAt: now.Unix(), ExpiresAt: expires.Unix(),
	})
	if err != nil {
		return LoginResponse{}, err
	}
	patch := client.MergeFrom(user.DeepCopy())
	user.Status.LastLoginAt = &metav1.Time{Time: now}
	_ = a.Client.Status().Patch(c, user, patch)

	secure := auth.HTTPS(c.Request)
	http.SetCookie(c.Writer, &http.Cookie{
		Name: SessionCookie, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
	device := now.Add(deviceLifetime)
	http.SetCookie(c.Writer, &http.Cookie{
		Name: DeviceCookie, Value: a.Signer.DeviceToken(user.Name, device), Path: deviceCookiePath, Expires: device,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
	return LoginResponse{Token: token, ExpiresAt: expires, User: a.userView(c, user)}, nil
}

// logout godoc
//
//	@Summary	Log out (ends the session and clears its cookie)
//	@Tags		Auth
//	@Success	204
//	@Router		/auth/logout [post]
func (a *API) logout(c *gin.Context) {
	if token := credentials(c); token != "" {
		a.Signer.Revoke(token, time.Now())
	}
	http.SetCookie(
		c.Writer,
		&http.Cookie{
			Name:     SessionCookie,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
	)
	c.Status(http.StatusNoContent)
}
