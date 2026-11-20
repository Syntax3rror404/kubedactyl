package httpapi

import (
	"net/http"
	"time"

	"app/base/api"
	"app/base/checks"
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
//	@Failure		404		{object}	api.ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers/{server}/diagnostics [get]
func (a *API) getServerDiagnostics(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	list := a.Diagnostics.Run(r.Context(), diagnostics.Input{
		Server: gs, Domain: a.externalDomain(w, r), ShowAddresses: api.PrincipalOf(r).Admin(),
	})
	api.JSON(w, http.StatusOK, ServerDiagnostics{Checks: list, CheckedAt: time.Now()})
}
