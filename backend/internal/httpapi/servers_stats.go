package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"app/api/v1alpha1"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
)

// ServerStats are live resource usage values.
type ServerStats struct {
	Phase v1alpha1.Phase `json:"phase" example:"Running"`
	// CPUMillis is the current CPU usage (1000 = one core); null when unknown.
	CPUMillis *int64 `json:"cpuMillis" extensions:"x-nullable" example:"350"`
	// MemoryBytes is the current memory usage; null when unknown.
	MemoryBytes *int64 `json:"memoryBytes" extensions:"x-nullable" example:"1073741824"`
	// DiskBytes is the space used on the data volume; null when unknown.
	DiskBytes *int64 `json:"diskBytes" extensions:"x-nullable" example:"524288000"`
	// UptimeSeconds since the game pod was started.
	UptimeSeconds int64 `json:"uptimeSeconds" example:"3600"`
}

// statsCache keeps live values shared by every viewer of a server: CPU and memory for 10 seconds (metrics-server
// measures every 15 seconds), the disk usage (a du in a pod) for 30 seconds or until the panel changed the files.
type statsCache struct {
	usage *kube.TTLCache[string, podUsage]
	disk  *kube.TTLCache[files.Ref, int64]
}

func newStatsCache() *statsCache {
	return &statsCache{
		usage: kube.NewTTLCache[string, podUsage](10 * time.Second),
		disk:  kube.NewTTLCache[files.Ref, int64](30 * time.Second),
	}
}

// podUsage is the CPU (millicores) and memory (bytes) of a game container; nil when unknown.
type podUsage struct {
	cpuMillis, memoryBytes *int64
}

// remeasureDisk makes the next stats request measure the disk usage again after a file operation that writes.
func (a *API) remeasureDisk(c *gin.Context) {
	c.Next()
	if gs, ok := c.Get(serverKey); ok && c.Request.Method != http.MethodGet {
		a.stats.disk.Forget(files.RefOf(gs.(*v1alpha1.GameServer)))
	}
}

var errNoPodForDisk = errors.New("no pod mounts the volume")

// diskUsage measures the server files in the file container or, while the server runs,
// in the game container. Without either pod the last value stored in the status is used.
func (a *API) diskUsage(ctx context.Context, gs *v1alpha1.GameServer) *int64 {
	b, err := a.stats.disk.Get(files.RefOf(gs), func() (int64, error) {
		switch {
		case a.Files.PodState(ctx, files.RefOf(gs)).Ready:
			return gameserver.DiskUsage(ctx, a.Kube, gs.Namespace, gameserver.FilesPodName(gs.Name),
				gameserver.FilesContainerName)
		case gs.Status.Phase == v1alpha1.PhaseRunning || gs.Status.Phase == v1alpha1.PhaseStarting:
			return gameserver.DiskUsage(ctx, a.Kube, gs.Namespace, gameserver.GamePodName(gs.Name),
				gameserver.ContainerName)
		}
		return 0, errNoPodForDisk
	})
	if err != nil {
		return gs.Status.DiskUsedBytes
	}
	return &b
}

type podMetrics struct {
	Containers []struct {
		Name  string            `json:"name"`
		Usage map[string]string `json:"usage"`
	} `json:"containers"`
}

// getServerStats godoc
//
//	@Summary		Live resource usage
//	@Description	CPU and memory from metrics-server (cached for 10 seconds), disk usage of the server files (cached
//	@Description	for 30 seconds; the last measured value while neither the file container nor the game runs).
//	@Tags			Servers
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Success		200		{object}	ServerStats
//	@Security		BearerAuth
//	@Router			/servers/{server}/stats [get]
func (a *API) getServerStats(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	st := ServerStats{Phase: gs.Status.Phase, DiskBytes: a.diskUsage(c, gs)}
	if gs.Status.PodUID != "" {
		if gs.Status.StartedAt != nil {
			st.UptimeSeconds = int64(time.Since(gs.Status.StartedAt.Time).Seconds())
		}
		st.CPUMillis, st.MemoryBytes = a.gamePodUsage(c, gs)
	}
	c.JSON(http.StatusOK, st)
}

// gamePodUsage reads CPU (millicores) and memory (bytes) of the game container from
// metrics-server; nil when metrics are not available (yet).
func (a *API) gamePodUsage(ctx context.Context, gs *v1alpha1.GameServer) (cpuMillis, memoryBytes *int64) {
	key := gs.Namespace + "/" + gs.Name + "/" + string(gs.Status.PodUID)
	u, _ := a.stats.usage.Get(key, func() (podUsage, error) {
		var u podUsage
		raw, err := a.Kube.Clientset.CoreV1().RESTClient().Get().
			AbsPath("/apis/metrics.k8s.io/v1beta1/namespaces", gs.Namespace, "pods", gameserver.GamePodName(gs.Name)).
			DoRaw(ctx)
		var m podMetrics
		if err != nil || json.Unmarshal(raw, &m) != nil {
			return u, nil // not measured yet: asked again after the TTL
		}
		for _, ct := range m.Containers {
			if ct.Name != gameserver.ContainerName {
				continue
			}
			if q, err := resource.ParseQuantity(ct.Usage["cpu"]); err == nil {
				u.cpuMillis = ptr.To(q.MilliValue())
			}
			if q, err := resource.ParseQuantity(ct.Usage["memory"]); err == nil {
				u.memoryBytes = ptr.To(q.Value())
			}
		}
		return u, nil
	})
	return u.cpuMillis, u.memoryBytes
}
