package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"app/app"
	"app/base/api"
	"app/internal/appsettings"
)

// RequestRates adds nothing: the footer of this app shows only the rates of the base.
func (a *API) RequestRates(*api.Principal) app.RequestRates { return app.RequestRates{} }

// StorageClassList lists the storage classes of the cluster.
type StorageClassList struct {
	Items []appsettings.StorageClass `json:"items"`
}

// PoolList lists the load balancer IP pools of the cluster.
type PoolList struct {
	Items []appsettings.Pool `json:"items"`
	// Error is set when the pools cannot be read (e.g. Cilium LB IPAM is not installed).
	Error string `json:"error,omitempty"`
	// IPv6Missing says why servers get no IPv6 address even from a pool with an IPv6 block
	// ("" when they can or it is unknown).
	IPv6Missing string `json:"ipv6Missing,omitempty" example:"No IPv6 service CIDR · IPv6 off in Cilium"`
}

// listStorageClasses godoc
//
//	@Summary	Storage classes of the cluster
//	@Tags		Settings
//	@Produce	json
//	@Success	200	{object}	StorageClassList
//	@Security	BearerAuth
//	@Router		/settings/storage-classes [get]
func (a *API) listStorageClasses(c *gin.Context) {
	classes, err := appsettings.ListStorageClasses(c, a.Reader)
	if err != nil {
		a.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, StorageClassList{Items: classes})
}

// listLoadBalancerPools godoc
//
//	@Summary		Load balancer IP pools of the cluster
//	@Description	Cilium LB IPAM pools with their address blocks, usage (also per IP family) and the service labels
//	@Description	that select them.
//	@Tags			Settings
//	@Produce		json
//	@Success		200	{object}	PoolList
//	@Security		BearerAuth
//	@Router			/settings/load-balancer-pools [get]
func (a *API) listLoadBalancerPools(c *gin.Context) {
	pools, err := appsettings.ListPoolsWithUsage(c, a.Reader)
	if errors.Is(err, appsettings.ErrNoCilium) {
		c.JSON(http.StatusOK, PoolList{Items: []appsettings.Pool{}, Error: err.Error()})
		return
	}
	if err != nil {
		a.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, PoolList{Items: pools, IPv6Missing: appsettings.IPv6Missing(c, a.Reader)})
}
