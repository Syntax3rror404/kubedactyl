package httpapi

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/settings"
	"app/internal/validation"
)

// UpdateServerRequest changes a game server; omitted fields stay unchanged.
// Runtime changes (image, startup, variables, memory, CPU) apply on the next start.
type UpdateServerRequest struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Image       *string           `json:"image,omitempty"`
	Startup     *string           `json:"startup,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	MemoryMiB   *int64            `json:"memoryMiB,omitempty"`
	CPUMillis   *int64            `json:"cpuMillis,omitempty"`
	DiskMiB     *int64            `json:"diskMiB,omitempty"`
	Ports       []int32           `json:"ports,omitempty"`
	// LoadBalancerPool moves the server to another enabled pool (its address changes).
	// Users may change it; a fixed IP is cleared unless a new one is given.
	LoadBalancerPool *string `json:"loadBalancerPool,omitempty"`
	LoadBalancerIP   *string `json:"loadBalancerIP,omitempty"`
	// ExternalTrafficPolicy of the service; admins only.
	ExternalTrafficPolicy *v1alpha1.TrafficPolicy `json:"externalTrafficPolicy,omitempty" binding:"omitempty,oneof=Local Cluster"`
	CrashRestart          *bool                   `json:"crashRestart,omitempty"`
	StopTimeoutSeconds    *int64                  `json:"stopTimeoutSeconds,omitempty"`
}

// updateServer godoc
//
//	@Summary		Update a game server
//	@Description	Runtime settings apply on the next start. Disk can only grow.
//	@Tags			Servers
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string				true	"Server name"
//	@Param			body	body		UpdateServerRequest	true	"Changes"
//	@Success		200		{object}	v1alpha1.GameServer
//	@Failure		422		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers/{server} [patch]
func (a *API) updateServer(c *gin.Context) {
	var req UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	e, ok := a.loadEgg(c, gs.Spec.EggRef)
	if !ok {
		return
	}
	p := principal(c)
	if !p.Admin() {
		if err := checkUserUpdate(&req, gs, e); err != nil {
			a.fail(c, err)
			return
		}
	}
	patch := client.MergeFrom(gs.DeepCopy())
	if err := a.applyServerUpdate(c, &req, gs, e); err != nil {
		a.fail(c, err)
		return
	}
	if err := a.Client.Patch(c, gs, patch); err != nil {
		a.fail(c, err)
		return
	}
	a.forgetServers(gs.Namespace)
	a.audit(c, "server updated", "server", gs.Name)
	a.Trigger(gs.Namespace, gs.Name)
	c.JSON(http.StatusOK, visibleServer(p, gs, e, a.externalDomain(c)))
}

// applyServerUpdate applies the fields of the request to the server spec and validates them.
func (a *API) applyServerUpdate(
	ctx context.Context, req *UpdateServerRequest, gs *v1alpha1.GameServer, e *v1alpha1.Egg,
) error {
	s := &gs.Spec
	applyGeneral(req, s)
	if err := applyEnvironment(req, s, e); err != nil {
		return err
	}
	if err := applyResources(req, s); err != nil {
		return err
	}
	return a.applyNetwork(ctx, req, s)
}

// applyGeneral sets name, image, startup, crash restart and stop timeout.
func applyGeneral(req *UpdateServerRequest, s *v1alpha1.GameServerSpec) {
	if req.DisplayName != nil && strings.TrimSpace(*req.DisplayName) != "" {
		s.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Image != nil && *req.Image != "" {
		s.Image = *req.Image
	}
	if req.Startup != nil {
		s.Startup = *req.Startup
	}
	if req.CrashRestart != nil {
		s.CrashRestart = req.CrashRestart
	}
	if req.StopTimeoutSeconds != nil {
		s.StopTimeoutSeconds = max(*req.StopTimeoutSeconds, 1)
	}
}

// applyEnvironment merges the given variables and checks all of them against the egg rules.
func applyEnvironment(req *UpdateServerRequest, s *v1alpha1.GameServerSpec, e *v1alpha1.Egg) error {
	if req.Environment == nil {
		return nil
	}
	if s.Environment == nil {
		s.Environment = map[string]string{}
	}
	maps.Copy(s.Environment, req.Environment)
	return gameserver.ValidateVariables(e, s.Environment)
}

// applyResources sets memory, CPU and disk (the volume can only grow) and the ports.
func applyResources(req *UpdateServerRequest, s *v1alpha1.GameServerSpec) error {
	if req.MemoryMiB != nil {
		s.Resources.MemoryMiB = *req.MemoryMiB
	}
	if req.CPUMillis != nil {
		s.Resources.CPUMillis = *req.CPUMillis
	}
	if req.DiskMiB != nil {
		if *req.DiskMiB < s.Resources.DiskMiB {
			return validation.Field("diskMiB", errors.New("disk size cannot be reduced"))
		}
		s.Resources.DiskMiB = *req.DiskMiB
	}
	if err := gameserver.ValidateResources(s.Resources); err != nil {
		return err
	}
	if req.Ports != nil {
		if err := gameserver.ValidatePorts(req.Ports); err != nil {
			return err
		}
		s.Ports = req.Ports
	}
	return nil
}

// applyNetwork moves the server to another enabled pool (clearing a fixed IP of the old pool),
// sets a fixed IP, which must lie inside the pool, and the external traffic policy.
func (a *API) applyNetwork(ctx context.Context, req *UpdateServerRequest, s *v1alpha1.GameServerSpec) error {
	if req.LoadBalancerPool != nil && *req.LoadBalancerPool != s.LoadBalancerPool {
		if *req.LoadBalancerPool == "" {
			return validation.Field("loadBalancerPool", errors.New("select a load balancer pool"))
		}
		set, err := a.Settings.Get(ctx)
		if err != nil {
			return err
		}
		_, err = settings.Choose(*req.LoadBalancerPool, "", set.LoadBalancerPools, "load balancer pool")
		if err != nil {
			return err
		}
		s.LoadBalancerPool = *req.LoadBalancerPool
		s.LoadBalancerIP = ""
	}
	if req.LoadBalancerIP != nil {
		s.LoadBalancerIP = *req.LoadBalancerIP
	}
	if req.ExternalTrafficPolicy != nil {
		s.ExternalTrafficPolicy = *req.ExternalTrafficPolicy
	}
	if req.LoadBalancerIP != nil || req.LoadBalancerPool != nil {
		if err := a.checkPoolIP(ctx, s.LoadBalancerPool, s.LoadBalancerIP); err != nil {
			return err
		}
	}
	return nil
}

// checkPoolIP verifies that a fixed IP is valid and inside the pool.
func (a *API) checkPoolIP(ctx context.Context, pool, ip string) error {
	if ip == "" {
		return nil
	}
	if net.ParseIP(ip) == nil {
		return validation.Field("loadBalancerIP", errors.New("invalid load balancer IP"))
	}
	if pool == "" {
		return nil
	}
	p, err := settings.GetPool(ctx, a.Reader, pool)
	if err != nil {
		return err
	}
	if !p.Contains(ip) {
		err := fmt.Errorf("%s is not in pool %s (%s)", ip, pool, strings.Join(p.Blocks, ", "))
		return validation.Field("loadBalancerIP", err)
	}
	return nil
}
