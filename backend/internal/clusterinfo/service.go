// Package clusterinfo collects information about the Kubernetes cluster for the admin
// "Cluster" page: nodes with hardware and usage, and the identity the panel connects with.
package clusterinfo

import (
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/internal/kube"
	"app/internal/settings"
)

// Service gathers cluster information on demand.
type Service struct {
	Kube *kube.Client
	// Config is the REST config the panel uses (for the identity summary).
	Config *rest.Config
	// Namespace is the panel namespace (probe pods and the hardware cache live there).
	Namespace string
	// HelperImage runs the hardware probe pods.
	HelperImage string
	// KubeContext is the explicitly selected kubeconfig context ("" = default).
	KubeContext string
	// HTTP1 reports whether the client talks HTTP/1.1 to the API server.
	HTTP1 bool
	// Reader (uncached) and Settings are used by the health checks.
	Reader   client.Reader
	Settings *settings.Store
	// SelfUpgrade adds the permission check for upgrade jobs.
	SelfUpgrade bool
	// AdmissionPolicy is the ValidatingAdmissionPolicy that limits the panel's cluster wide
	// permissions (chart: admission-policy.yaml; "" when the cluster has no such API).
	AdmissionPolicy string
	Log             *slog.Logger

	mu       sync.Mutex
	hardware map[string]*hardwareEntry
	loaded   bool

	// Shared by every viewer (the sidebar, dashboard and cluster page of each administrator ask the same);
	// created on first use.
	cachesOnce      sync.Once
	healthCache     *kube.TTLCache[struct{}, *Health]
	permissionCache *kube.TTLCache[struct{}, []Permission]
	nodeCache       *kube.TTLCache[struct{}, *nodeData]
}

// Cache lifetimes: the permissions change only with the chart (an upgrade restarts the panel); node lists and
// metrics are polled every 10 seconds by the cluster page (metrics-server measures every 15 seconds).
const (
	healthTTL     = 30 * time.Second
	permissionTTL = 10 * time.Minute
	nodeTTL       = 10 * time.Second
)

func (s *Service) caches() *Service {
	s.cachesOnce.Do(func() {
		s.healthCache = kube.NewTTLCache[struct{}, *Health](healthTTL)
		s.permissionCache = kube.NewTTLCache[struct{}, []Permission](permissionTTL)
		s.nodeCache = kube.NewTTLCache[struct{}, *nodeData](nodeTTL)
	})
	return s
}

type hardwareEntry struct {
	Hardware *Hardware `json:"hardware,omitempty"`
	Error    string    `json:"error,omitempty"`
	ProbedAt time.Time `json:"probedAt"`
	probing  bool
}
