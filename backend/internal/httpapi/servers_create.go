package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/settings"
	"app/internal/tenancy"
	"app/internal/validation"
)

// CreateServerRequest creates a game server from an egg.
type CreateServerRequest struct {
	DisplayName string `json:"displayName"           binding:"required" example:"Survival"`
	Egg         string `json:"egg"                   binding:"required" example:"paper"`
	Image       string `json:"image,omitempty"                          example:"ghcr.io/pelican-eggs/yolks:java_21"`
	Startup     string `json:"startup,omitempty"`
	// StartupName picks one of the egg's startup commands ("" = its default).
	StartupName string            `json:"startupName,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	MemoryMiB   int64             `json:"memoryMiB"             binding:"required" example:"4096"`
	CPUMillis   int64             `json:"cpuMillis,omitempty"                      example:"2000"`
	DiskMiB     int64             `json:"diskMiB"               binding:"required" example:"10240"`
	Ports       []int32           `json:"ports"                 binding:"required" example:"25565"`
	// StorageClass of the data volume (default: the default of the panel settings).
	StorageClass string `json:"storageClass,omitempty" example:"longhorn"`
	// LoadBalancerPool the address comes from (default: the default of the panel settings).
	LoadBalancerPool string `json:"loadBalancerPool,omitempty" example:"general-pool"`
	// LoadBalancerIP optionally requests a fixed IP from the pool (one per IP family, separated by a comma).
	LoadBalancerIP string `json:"loadBalancerIP,omitempty" example:"192.168.1.70"`
	// ExternalTrafficPolicy of the service (default Local: the server sees the players' IPs).
	ExternalTrafficPolicy v1alpha1.TrafficPolicy `json:"externalTrafficPolicy,omitempty" binding:"omitempty,oneof=Local Cluster"`
	// IPv6 also asks for an IPv6 address (dual stack clusters).
	IPv6 bool `json:"ipv6,omitempty"`
	// Owner is the username the server belongs to (default: the calling admin).
	Owner string `json:"owner,omitempty" example:"alice"`
	// StartOnCompletion starts the server once the installation is finished.
	StartOnCompletion bool `json:"startOnCompletion,omitempty" example:"true"`
	SkipInstall       bool `json:"skipInstall,omitempty"`
}

// createServer godoc
//
//	@Summary		Create a game server
//	@Description	Creates the server and runs the egg install script. Variables are validated against the egg rules.
//	@Tags			Servers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateServerRequest	true	"Server"
//	@Success		201		{object}	v1alpha1.GameServer
//	@Failure		422		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers [post]
func (a *API) createServer(c *gin.Context) {
	var req CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	e, ok := a.loadEgg(c, req.Egg)
	if !ok {
		return
	}
	if req.Owner == "" {
		req.Owner = principal(c).User.Name
	}
	gs, err := a.newGameServer(c, &req, e)
	if err != nil {
		a.fail(c, err)
		return
	}
	if err := a.Client.Create(c, gs); err != nil {
		a.fail(c, err)
		return
	}
	a.Trigger(gs.Namespace, gs.Name)
	a.audit(c, "server created", "server", gs.Name, "egg", gs.Spec.EggRef, "namespace", gs.Namespace)
	c.JSON(http.StatusCreated, gs)
}

// newGameServer validates a create request and builds the GameServer object: resources and
// ports, storage class and pool (defaults from the panel settings), variables (egg defaults,
// checked against the egg rules), the owner's namespace and a free server name.
func (a *API) newGameServer(
	ctx context.Context, req *CreateServerRequest, e *v1alpha1.Egg,
) (*v1alpha1.GameServer, error) {
	if req.Image == "" {
		req.Image = e.Spec.DockerImages[0].Image
	}
	resources := v1alpha1.Resources{MemoryMiB: req.MemoryMiB, CPUMillis: req.CPUMillis, DiskMiB: req.DiskMiB}
	if err := gameserver.ValidateResources(resources); err != nil {
		return nil, err
	}
	if err := gameserver.ValidatePorts(req.Ports); err != nil {
		return nil, err
	}
	if err := gameserver.ValidateStartupName(e, req.StartupName); err != nil {
		return nil, err
	}
	storageClass, pool, err := a.choosePlacement(ctx, req)
	if err != nil {
		return nil, err
	}
	env := serverEnvironment(e, req.Environment)
	if err := gameserver.ValidateVariables(e, env); err != nil {
		return nil, err
	}
	namespace, err := a.ownerNamespace(ctx, req.Owner)
	if err != nil {
		return nil, err
	}
	name, err := a.freeServerName(ctx, req.DisplayName)
	if err != nil {
		return nil, err
	}
	state := v1alpha1.PowerStopped
	if req.StartOnCompletion {
		state = v1alpha1.PowerRunning
	}
	return &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: namespace, Labels: map[string]string{tenancy.LabelOwner: req.Owner},
		},
		Spec: v1alpha1.GameServerSpec{
			DisplayName:           req.DisplayName,
			EggRef:                e.Name,
			Image:                 req.Image,
			Startup:               req.Startup,
			StartupName:           req.StartupName,
			Environment:           env,
			Resources:             resources,
			Ports:                 req.Ports,
			StorageClass:          storageClass,
			LoadBalancerPool:      pool,
			LoadBalancerIP:        req.LoadBalancerIP,
			ExternalTrafficPolicy: req.ExternalTrafficPolicy,
			IPv6:                  req.IPv6,
			State:                 state,
			InstallRevision:       1,
			SkipInstall:           req.SkipInstall,
			CrashRestart:          ptr.To(true),
			StopTimeoutSeconds:    600,
		},
	}, nil
}

// choosePlacement picks storage class and load balancer pool (the panel defaults when the
// request names none) and checks a fixed IP against the pool.
func (a *API) choosePlacement(ctx context.Context, req *CreateServerRequest) (storageClass, pool string, err error) {
	set, err := a.Settings.Get(ctx)
	if err != nil {
		return "", "", err
	}
	storageClass, err = settings.Choose(req.StorageClass, set.DefaultStorageClass, set.StorageClasses, "storage class")
	if err == nil && storageClass == "" {
		err = settings.ErrNotConfigured
	}
	if err != nil {
		return "", "", err
	}
	pool, err = settings.Choose(
		req.LoadBalancerPool, set.DefaultLoadBalancerPool, set.LoadBalancerPools, "load balancer pool",
	)
	if err != nil {
		return "", "", err
	}
	if err := a.checkPoolIP(ctx, pool, req.LoadBalancerIP); err != nil {
		return "", "", err
	}
	return storageClass, pool, nil
}

// serverEnvironment returns a value for every egg variable: the requested one or the default.
func serverEnvironment(e *v1alpha1.Egg, requested map[string]string) map[string]string {
	env := map[string]string{}
	for _, v := range e.Spec.Variables {
		if val, ok := requested[v.EnvVariable]; ok {
			env[v.EnvVariable] = val
		} else {
			env[v.EnvVariable] = v.DefaultValue
		}
	}
	return env
}

// ownerNamespace checks that the owner exists and returns (creates) its namespace.
func (a *API) ownerNamespace(ctx context.Context, owner string) (string, error) {
	u := &v1alpha1.User{}
	err := a.Reader.Get(ctx, client.ObjectKey{Namespace: a.Opts.Namespace, Name: owner}, u)
	if apierrors.IsNotFound(err) {
		return "", validation.Field("owner", fmt.Errorf("owner %q does not exist", owner))
	}
	if err != nil {
		return "", err
	}
	return tenancy.EnsureNamespace(ctx, a.Client, owner)
}

// freeServerName returns a server name derived from the display name that no other server
// uses (names are unique across all namespaces: the API addresses servers by name).
func (a *API) freeServerName(ctx context.Context, displayName string) (string, error) {
	for range 5 {
		candidate := gameserver.NewName(displayName)
		taken, err := a.serverNameTaken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", conflict(errors.New("could not find a free server name, try again"))
}
