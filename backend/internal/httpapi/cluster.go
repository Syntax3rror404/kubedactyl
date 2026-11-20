package httpapi

import (
	"context"
	"net/http"
	"time"

	"app/base/api"
)

// getClusterInfo godoc
//
//	@Summary	Cluster information
//	@Tags		Cluster
//	@Produce	json
//	@Success	200	{object}	ClusterInfo
//	@Security	BearerAuth
//	@Router		/cluster [get]
func (a *API) getClusterInfo(w http.ResponseWriter, r *http.Request) {
	info := a.Cluster
	if set, err := a.Settings.Current(r.Context()); err == nil {
		info.StorageClass, info.LoadBalancerPool = set.DefaultStorageClass, set.DefaultLoadBalancerPool
	}
	api.JSON(w, http.StatusOK, info)
}

// listClusterNodes godoc
//
//	@Summary		Cluster nodes
//	@Description	Nodes with roles, status, capacity, requested resources, live usage (metrics-server) and hardware
//	@Description	(CPU model, vendor, product) plus totals. Ready nodes without hardware data are probed in the
//	@Description	background.
//	@Tags			Cluster
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	clusterinfo.NodesResponse
//	@Failure		403	{object}	api.ErrorResponse
//	@Router			/cluster/nodes [get]
func (a *API) listClusterNodes(w http.ResponseWriter, r *http.Request) {
	resp, err := a.ClusterService.Nodes(r.Context())
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	api.JSON(w, http.StatusOK, resp)
}

// probeClusterNodes godoc
//
//	@Summary		Read the hardware of all nodes again
//	@Description	Starts a short-lived, unprivileged probe pod on every ready node that reads /proc/cpuinfo and
//	@Description	/sys/class/dmi. Waits for the results (up to 90 seconds).
//	@Tags			Cluster
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	clusterinfo.NodesResponse
//	@Router			/cluster/nodes/probe [post]
func (a *API) probeClusterNodes(w http.ResponseWriter, r *http.Request) {
	resp, err := a.ClusterService.Nodes(r.Context())
	if err != nil {
		a.Fail(w, r, err)
		return
	}
	var nodes []string
	for _, n := range resp.Nodes {
		if n.Ready && !n.Unschedulable {
			nodes = append(nodes, n.Name)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	a.ClusterService.ProbeNodes(ctx, nodes)
	if resp, err = a.ClusterService.Nodes(r.Context()); err != nil {
		a.Fail(w, r, err)
		return
	}
	api.JSON(w, http.StatusOK, resp)
}

// getClusterIdentity godoc
//
//	@Summary		Connection and identity of the panel
//	@Description	Kubeconfig or service account in use, authentication method (secrets are never returned), TLS, the
//	@Description	identity reported by the API server (SelfSubjectReview) and the permissions the panel needs.
//	@Tags			Cluster
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	clusterinfo.Identity
//	@Router			/cluster/identity [get]
func (a *API) getClusterIdentity(w http.ResponseWriter, r *http.Request) {
	api.JSON(w, http.StatusOK, a.ClusterService.Identity(r.Context()))
}

// getClusterHealth godoc
//
//	@Summary		Health of the panel's dependencies
//	@Description	CRDs, permissions, nodes, metrics-server, load balancer pools and storage classes (cached for 30
//	@Description	seconds).
//	@Tags			Cluster
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	clusterinfo.Health
//	@Router			/cluster/health [get]
func (a *API) getClusterHealth(w http.ResponseWriter, r *http.Request) {
	api.JSON(w, http.StatusOK, a.ClusterService.Health(r.Context()))
}
