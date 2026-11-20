package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/auth"
	"app/base/validation"
)

// loginLimit is a limiter of failed sign-ins and the key a sign-in counts under.
type loginLimit struct {
	limiter *auth.Limiter
	key     string
}

// DeviceCookie marks a browser in which the user signed in before (path deviceCookiePath only): its sign-ins
// do not count against the account, so failed attempts from elsewhere cannot lock the user out.
const (
	DeviceCookie     = app.CookiePrefix + "device"
	deviceCookiePath = "/api/auth/login"
	deviceLifetime   = 180 * 24 * time.Hour
)

// loginLimits are the limits of a sign-in: client and user (reset by a success), the client
// alone (password spraying) and the user alone (from many addresses or a spoofed X-Forwarded-For),
// except from a browser the user signed in with before.
func (a *API) loginLimits(r *http.Request, clientIP, username string, now time.Time) []loginLimit {
	limits := []loginLimit{{a.Limiter, clientIP + "|" + username}, {a.ClientLimiter, clientIP}}
	if device, err := r.Cookie(DeviceCookie); err != nil || !a.Signer.KnownDevice(device.Value, username, now) {
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
//
// The maximum lengths (usernames have at most 32 characters, auth.MaxPasswordLength) keep a request from filling
// the limiters and the log with huge keys.
type LoginRequest struct {
	Username string `json:"username" maxLength:"64"   example:"admin"`
	Password string `json:"password" maxLength:"1024" example:"secret-password"`
}

// Validate checks that both fields are set and not too long.
func (req LoginRequest) Validate() error {
	errs := validation.Errors{}
	errs.Require("username", req.Username != "")
	errs.MaxLength("username", req.Username, 64)
	errs.Require("password", req.Password != "")
	errs.MaxLength("password", req.Password, 1024)
	return errs.OrNil()
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
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !a.Decode(w, r, &req) {
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	// ClientIP honors X-Forwarded-For only from trusted proxies (--trusted-proxies).
	clientIP := a.Proxies.ClientIP(r)
	now := time.Now()
	limits := a.loginLimits(r, clientIP, username, now)
	if wait, ok := takeAttempt(limits, now); !ok {
		retryAfter(w, wait)
		a.Fail(w, r, TooManyRequests(errors.New("too many failed logins, try again later")))
		return
	}
	user, ok := a.checkPassword(r.Context(), username, req.Password)
	if !ok {
		a.Log.Warn("sign-in failed", "user", username, "ip", clientIP)
		a.Fail(w, r, Unauthorized(errors.New("invalid username or password")))
		return
	}
	a.Limiter.Reset(limits[0].key)
	forgiveAttempt(limits[1:], now)
	a.Log.Info("signed in", "user", user.Name, "ip", clientIP)
	if err := a.Users.UpdatePasswordHash(r.Context(), user, req.Password); err != nil {
		a.Log.Warn("updating the password hash", "user", user.Name, "err", err)
	}
	a.sendSession(w, r, http.StatusOK, user)
}

// checkPassword returns the user when the password matches and the account is active (users.Store.CheckActive).
// The password is checked for unknown, disabled and single sign-on only users, too: the answer takes as long
// either way.
func (a *API) checkPassword(ctx context.Context, username, password string) (*v1alpha1.User, bool) {
	user := &v1alpha1.User{}
	err := a.Client.Get(ctx, client.ObjectKey{Namespace: a.Namespace, Name: username}, user)
	if err != nil || user.Spec.PasswordHash == "" {
		auth.VerifyDummy(password)
		return nil, false
	}
	return user, auth.VerifyPassword(password, user.Spec.PasswordHash) && a.Users.CheckActive(ctx, user) == nil
}

// startSession signs a session token for the user and sets it as cookie.
func (a *API) startSession(w http.ResponseWriter, r *http.Request, user *v1alpha1.User) (LoginResponse, error) {
	now := time.Now()
	expires := now.Add(a.Settings.Lifetimes(r.Context()).Session)
	token, err := a.Signer.Sign(auth.Session{
		User: user.Name, Epoch: user.Spec.SessionEpoch, IssuedAt: now.Unix(), ExpiresAt: expires.Unix(),
	})
	if err != nil {
		return LoginResponse{}, err
	}
	patch := client.MergeFrom(user.DeepCopy())
	user.Status.LastLoginAt = &metav1.Time{Time: now}
	_ = a.Client.Status().Patch(r.Context(), user, patch)

	secure := auth.HTTPS(r)
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName(r), Value: token, Path: "/", Expires: expires,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
	device := now.Add(deviceLifetime)
	http.SetCookie(w, &http.Cookie{
		Name: DeviceCookie, Value: a.Signer.DeviceToken(user.Name, device), Path: deviceCookiePath, Expires: device,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
	return LoginResponse{Token: token, ExpiresAt: expires, User: a.userView(r.Context(), user)}, nil
}

// sendSession signs the user in and answers with the session.
func (a *API) sendSession(w http.ResponseWriter, r *http.Request, status int, user *v1alpha1.User) {
	res, err := a.startSession(w, r, user)
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	JSON(w, status, res)
}

// logout godoc
//
//	@Summary		Log out (ends the session and clears its cookie)
//	@Description	A user who logs out very often in one session lifetime is signed out everywhere instead.
//	@Tags			Auth
//	@Success		204
//	@Router			/auth/logout [post]
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if token := Credentials(r); token != "" {
		a.revoke(r.Context(), token)
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName(r), Path: "/", MaxAge: -1, HttpOnly: true, Secure: auth.HTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// revoke ends a session token: its user keeps it as logged out until it would have expired (users.Store.LogOut).
func (a *API) revoke(ctx context.Context, token string) {
	sess, err := a.Signer.Verify(token, time.Now())
	if err != nil {
		return
	}
	if err := a.Users.LogOut(ctx, sess.User, auth.SessionID(token), time.Unix(sess.ExpiresAt, 0)); err != nil {
		a.Log.Warn("logging out", "user", sess.User, "err", err)
	}
}
