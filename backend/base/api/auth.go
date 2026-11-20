package api

import (
	"context"
	"errors"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/auth"
	"app/base/users"
)

// SessionCookie holds the signed session token of the web UI. Over HTTPS it is a __Host- cookie (path /, no
// domain), which a site on a sibling subdomain cannot set.
const SessionCookie = app.CookiePrefix + "session"

// sessionCookieName is the name of the session cookie for the request (see SessionCookie).
func sessionCookieName(r *http.Request) string {
	if auth.HTTPS(r) {
		return "__Host-" + SessionCookie
	}
	return SessionCookie
}

// principalKey holds the Principal in the context of an authenticated request.
type principalKey struct{}

// Principal is the authenticated caller.
type Principal struct {
	User *v1alpha1.User
	// TokenID is set when the request used an API token.
	TokenID string
}

// Admin reports whether the caller may do everything.
func (p *Principal) Admin() bool { return p.User.Spec.Role == v1alpha1.RoleAdmin }

// PrincipalOf is the caller of a request that passed authenticate (nil before).
func PrincipalOf(r *http.Request) *Principal {
	p, _ := r.Context().Value(principalKey{}).(*Principal)
	return p
}

var (
	errUnauthorized    = errors.New("authentication required")
	errTooManyAttempts = errors.New("too many attempts, try again later")
)

// crossOrigin rejects state changing requests from other origins (Sec-Fetch-Site, else Origin against Host).
var crossOrigin http.CrossOriginProtection

// crossSiteCookieRequest detects state changing requests that another site triggered
// with the session cookie (CSRF). The SameSite=Strict cookie already prevents this in
// current browsers; the origin check is a second, proxy independent one. Requests with
// a bearer token (API tokens, scripts) are not affected: they do not use the cookie.
func crossSiteCookieRequest(r *http.Request) bool {
	if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return false
	}
	return crossOrigin.Check(r) != nil
}

// sameOrigin guards the public sign-in and setup against other sites (login CSRF: signing the
// victim into the attacker's account): the origin check, and a JSON body, which a form of another
// site cannot send without a CORS preflight.
func sameOrigin(r *http.Request) error {
	if t, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || t != "application/json" {
		return Forbidden(errCrossSite)
	}
	return notCrossSite(r)
}

var errCrossSite = errors.New("cross-site request rejected")

// notCrossSite refuses a request with the session cookie from another site, also a sibling subdomain.
func notCrossSite(r *http.Request) error {
	if crossSiteCookieRequest(r) {
		return Forbidden(errCrossSite)
	}
	return nil
}

// Credentials returns the bearer token or the session cookie of a request.
func Credentials(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if c, err := r.Cookie(sessionCookieName(r)); err == nil {
		return c.Value
	}
	return ""
}

// authenticate resolves session tokens and API tokens to a principal.
func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := Credentials(r)
		if token == "" {
			a.Fail(w, r, Unauthorized(errUnauthorized))
			return
		}
		if err := notCrossSite(r); err != nil {
			a.Fail(w, r, err)
			return
		}
		p, err := a.PrincipalFor(r.Context(), token)
		if err != nil {
			a.Fail(w, r, Unauthorized(errUnauthorized))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}

// limitRequests refuses a request of a user who sent too many, or while the panel's Kubernetes
// API calls queue up (settings "Kube API limit"), before the handler starts: an admitted request
// always runs to its end. Retry-After says when to try again. Administrators wait in the queue
// instead of being refused as busy: they must be able to raise the limit during an overload.
func (a *API) limitRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wait, err := a.Admit(PrincipalOf(r)); err != nil {
			retryAfter(w, wait)
			a.Fail(w, r, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// retryAfter tells the client when to try again, in whole seconds (at least 1) as the header requires.
func retryAfter(w http.ResponseWriter, wait time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(max(1, int(math.Ceil(wait.Seconds())))))
}

// takeClientAttempt counts an attempt of the client at a public route under kind ("setup", "invite", "oidc")
// before it runs, like a sign-in: the caller forgives it (Limiter.Forgive with the key and now) unless it
// failed. A refusal sets Retry-After and returns the error to answer with.
func (a *API) takeClientAttempt(w http.ResponseWriter, r *http.Request, kind string, now time.Time) (string, error) {
	key := kind + "|" + a.Proxies.ClientIP(r)
	if ok, wait := a.Limiter.Take(key, now); !ok {
		retryAfter(w, wait)
		return key, TooManyRequests(errTooManyAttempts)
	}
	return key, nil
}

// Admit counts one request of the user (also every message of an open websocket) and refuses it over the
// user's limit or, for non-administrators, while the panel is busy.
func (a *API) Admit(p *Principal) (time.Duration, error) {
	if a.KubeLimiter == nil {
		return 0, nil
	}
	if !p.Admin() {
		if wait, err := a.KubeLimiter.Busy(); err != nil {
			return wait, err
		}
	}
	return a.KubeLimiter.Admit(p.User.Name)
}

// PrincipalFor resolves a session or API token of an active user (users.Store.CheckActive).
func (a *API) PrincipalFor(ctx context.Context, token string) (*Principal, error) {
	user, tokenID, err := a.resolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := a.Users.CheckActive(ctx, user); err != nil {
		return nil, auth.ErrInvalidToken
	}
	return &Principal{User: user, TokenID: tokenID}, nil
}

func (a *API) resolveToken(ctx context.Context, token string) (*v1alpha1.User, string, error) {
	now := time.Now()
	lifetimes := a.Settings.Lifetimes(ctx)
	if id := auth.APITokenID(token); id != "" {
		return a.resolveAPIToken(ctx, token, id, now, lifetimes.APIToken)
	}
	sess, err := a.Signer.Verify(token, now)
	if err != nil {
		return nil, "", err
	}
	// Sessions signed before the lifetime was shortened end with it, too.
	if sess.IssuedAt != 0 && !now.Before(time.Unix(sess.IssuedAt, 0).Add(lifetimes.Session)) {
		return nil, "", auth.ErrInvalidToken
	}
	user, err := a.sessionUser(ctx, sess)
	if err == nil && users.LoggedOut(user, auth.SessionID(token)) {
		return nil, "", auth.ErrInvalidToken
	}
	return user, "", err
}

// sessionUser reads the user of a signed session. The cache can lag behind a change made a moment
// ago (an account created through an invite, a new session epoch), so a missing user or an older
// epoch is read again from the API server, only for sessions with a valid signature.
func (a *API) sessionUser(ctx context.Context, sess auth.Session) (*v1alpha1.User, error) {
	key := client.ObjectKey{Namespace: a.Namespace, Name: sess.User}
	user := &v1alpha1.User{}
	err := a.Client.Get(ctx, key, user)
	if apierrors.IsNotFound(err) || (err == nil && user.Spec.SessionEpoch < sess.Epoch) {
		err = a.Reader.Get(ctx, key, user)
	}
	if err != nil {
		return nil, err
	}
	if user.Spec.SessionEpoch != sess.Epoch {
		return nil, auth.ErrInvalidToken
	}
	return user, nil
}

// resolveAPIToken finds the user of an API token that is neither expired nor older than maxAge.
func (a *API) resolveAPIToken(
	ctx context.Context, token, id string, now time.Time, maxAge time.Duration,
) (*v1alpha1.User, string, error) {
	var users v1alpha1.UserList
	if err := a.Client.List(ctx, &users, client.InNamespace(a.Namespace)); err != nil {
		return nil, "", err
	}
	for i := range users.Items {
		for _, t := range users.Items[i].Spec.Tokens {
			if t.ID == id && auth.MatchToken(token, t.Hash) && now.Before(tokenExpiry(t, maxAge)) {
				return &users.Items[i], id, nil
			}
		}
	}
	return nil, "", auth.ErrInvalidToken
}

// requireAdmin rejects non-admin callers.
func requireAdmin(r *http.Request) error {
	if p := PrincipalOf(r); p == nil || !p.Admin() {
		return Forbidden(errors.New("administrator permission required"))
	}
	return nil
}

var errOwnPassword = errors.New("choose your own password first")

// requireOwnPassword rejects users who still have to replace the password an administrator set
// (sessions and API tokens alike); they may only read their account, change the password and sign out.
func requireOwnPassword(r *http.Request) error {
	if PrincipalOf(r).User.Spec.MustChangePassword {
		return Forbidden(errOwnPassword)
	}
	return nil
}
