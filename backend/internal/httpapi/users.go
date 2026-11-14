package httpapi

import (
	"context"
	"slices"

	"github.com/gin-gonic/gin"

	"app/api/v1alpha1"
	"app/app"
	"app/base/api"
	"app/internal/appusers"
	"app/internal/tenancy"
)

// The app's parts of the account and invite answers of the base API (api.App).

// UserViews adds the owner of a subuser, the namespace and the number of servers of each account. All servers
// are listed once; a failed list counts none.
func (a *API) UserViews(ctx context.Context, list []*v1alpha1.User) []app.UserView {
	var servers v1alpha1.GameServerList
	_ = a.Client.List(ctx, &servers)
	out := make([]app.UserView, len(list))
	for i, u := range list {
		out[i] = app.UserView{Owner: u.Spec.Owner, Servers: serverCount(u, servers.Items)}
		if !appusers.Subuser(u) {
			out[i].Namespace = tenancy.Namespace(u.Name)
		}
	}
	return out
}

// serverCount counts the servers of a user, for a subuser the servers of its owner shared with it.
func serverCount(u *v1alpha1.User, servers []v1alpha1.GameServer) int {
	shared := func(s v1alpha1.ServerShare) bool { return s.User == u.Name }
	n := 0
	for _, gs := range servers {
		visible := !appusers.Subuser(u) || slices.ContainsFunc(gs.Spec.SharedWith, shared)
		if gs.Namespace == appusers.Home(u) && visible {
			n++
		}
	}
	return n
}

// InviteView adds the owner of an invite for a subuser and the servers it shares.
func (a *API) InviteView(i *v1alpha1.Invite) app.InviteView {
	return app.InviteView{Owner: i.Spec.Owner, Shares: append([]v1alpha1.InviteShare{}, i.Spec.Shares...)}
}

// requireNoSubuser rejects subusers: they cannot have subusers of their own.
func (a *API) requireNoSubuser(c *gin.Context) {
	if subuser(api.PrincipalOf(c)) {
		a.Fail(c, api.Forbidden(appusers.ErrSubuser))
		return
	}
	c.Next()
}
