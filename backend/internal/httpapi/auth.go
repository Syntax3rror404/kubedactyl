package httpapi

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/tenancy"
)

// SessionCookie holds the signed session token of the web UI.
const SessionCookie = "kd_session"

const principalKey = "kubedactyl.principal"

// Principal is the authenticated caller.
type Principal struct {
	User *v1alpha1.User
	// Namespace owns the game servers of the user.
	Namespace string
	// TokenID is set when the request used an API token.
	TokenID string
}

// Admin reports whether the caller may do everything.
func (p *Principal) Admin() bool { return p.User.Spec.Role == v1alpha1.RoleAdmin }

func principal(c *gin.Context) *Principal {
	p, _ := c.Get(principalKey)
	pr, _ := p.(*Principal)
	return pr
}

var errUnauthorized = errors.New("authentication required")

// crossSiteCookieRequest detects state changing requests that another site triggered
// with the session cookie (CSRF). The SameSite=Strict cookie already prevents this in
// current browsers; Sec-Fetch-Site is a second, proxy independent check. Requests with
// an Authorization header (API tokens, scripts) are not affected.
func crossSiteCookieRequest(c *gin.Context) bool {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	if c.GetHeader("Authorization") != "" {
		return false
	}
	site := c.GetHeader("Sec-Fetch-Site")
	return site != "" && site != "same-origin" && site != "none"
}

// sameOrigin guards the public sign-in and setup against other sites (login CSRF: signing the
// victim into the attacker's account): Sec-Fetch-Site, and a JSON body, which a form of another
// site cannot send without a CORS preflight.
func (a *API) sameOrigin(c *gin.Context) {
	if crossSiteCookieRequest(c) || c.ContentType() != "application/json" {
		a.fail(c, forbidden(errors.New("cross-site request rejected")))
		return
	}
	c.Next()
}

// credentials returns the bearer token or the session cookie of a request.
func credentials(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if v, err := c.Cookie(SessionCookie); err == nil {
		return v
	}
	return ""
}

// authenticate resolves session tokens and API tokens to a principal.
func (a *API) authenticate(c *gin.Context) {
	token := credentials(c)
	if token == "" {
		a.fail(c, unauthorized(errUnauthorized))
		return
	}
	if crossSiteCookieRequest(c) {
		a.fail(c, forbidden(errors.New("cross-site request rejected")))
		return
	}
	p, err := a.principalFor(c, token)
	if err != nil {
		a.fail(c, unauthorized(errUnauthorized))
		return
	}
	c.Set(principalKey, p)
	c.Next()
}

// limitRequests refuses a request of a user who sent too many, or while the panel's Kubernetes
// API calls queue up (settings "Kube API limit"), before the handler starts: an admitted request
// always runs to its end. Retry-After says when to try again. Administrators wait in the queue
// instead of being refused as busy: they must be able to raise the limit during an overload.
func (a *API) limitRequests(c *gin.Context) {
	if wait, err := a.admit(principal(c)); err != nil {
		c.Header("Retry-After", strconv.Itoa(max(1, int(math.Ceil(wait.Seconds())))))
		a.fail(c, err)
	}
}

// admit counts one request of the user (also every message of an open console) and refuses it over the
// user's limit or, for non-administrators, while the panel is busy.
func (a *API) admit(p *Principal) (time.Duration, error) {
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

// principalFor resolves a session or API token of an enabled user.
func (a *API) principalFor(ctx context.Context, token string) (*Principal, error) {
	user, tokenID, err := a.resolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if user.Spec.Disabled {
		return nil, auth.ErrInvalidToken
	}
	return &Principal{User: user, Namespace: tenancy.Namespace(user.Name), TokenID: tokenID}, nil
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
	return user, "", err
}

// sessionUser reads the user of a signed session. The cache can lag behind a change made a moment
// ago (an account created through an invite, a new session epoch), so a missing user or an older
// epoch is read again from the API server, only for sessions with a valid signature.
func (a *API) sessionUser(ctx context.Context, sess auth.Session) (*v1alpha1.User, error) {
	key := client.ObjectKey{Namespace: a.Opts.Namespace, Name: sess.User}
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
	if err := a.Client.List(ctx, &users, client.InNamespace(a.Opts.Namespace)); err != nil {
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
func (a *API) requireAdmin(c *gin.Context) {
	if p := principal(c); p == nil || !p.Admin() {
		a.fail(c, forbidden(errors.New("administrator permission required")))
		return
	}
	c.Next()
}

var errOwnPassword = errors.New("choose your own password first")

// requireOwnPassword rejects users who still have to replace the password an administrator set
// (sessions and API tokens alike); they may only read their account, change the password and sign out.
func (a *API) requireOwnPassword(c *gin.Context) {
	if principal(c).User.Spec.MustChangePassword {
		a.fail(c, forbidden(errOwnPassword))
		return
	}
	c.Next()
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
func (a *API) getMe(c *gin.Context) {
	c.JSON(http.StatusOK, a.userView(c, principal(c).User))
}
