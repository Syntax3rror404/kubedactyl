package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"app/internal/checks"
	"app/internal/diagnostics"
)

// ServerDiagnostics are the checks why a server may not be reachable.
type ServerDiagnostics struct {
	Checks    []checks.Check `json:"checks"`
	CheckedAt time.Time      `json:"checkedAt"`
}

// getServerDiagnostics godoc
//
//	@Summary		Diagnostics of a server
//	@Description	Checks the typical causes why players cannot connect: installation, state, pending restart, disk
//	@Description	space, load balancer address, DNS A/AAAA records of the external domain and the router forwarding
//	@Description	(TCP, tested from inside the network). Users do not see the load balancer IP.
//	@Tags			Servers
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Success		200		{object}	ServerDiagnostics
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers/{server}/diagnostics [get]
func (a *API) getServerDiagnostics(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	list := a.Diagnostics.Run(c, diagnostics.Input{
		Server: gs, Domain: a.externalDomain(c), ShowAddresses: principal(c).Admin(),
	})
	c.JSON(http.StatusOK, ServerDiagnostics{Checks: list, CheckedAt: time.Now()})
}
