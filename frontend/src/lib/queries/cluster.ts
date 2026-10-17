import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { NodesResponse } from "@/lib/types"

/** Connection summary of the sidebar cluster card. */
export const useClusterInfo = (enabled = true) =>
  useQuery({ queryKey: keys.cluster, queryFn: api.cluster.getInfo, staleTime: 60_000, enabled })

/** The nodes with capacity, usage and hardware, polled for the cluster page. */
export const useClusterNodes = () =>
  useQuery({
    queryKey: keys.clusterNodes,
    queryFn: api.cluster.listNodes,
    // Faster while hardware probes run, so their results show soon after they finish.
    refetchInterval: (q) => (q.state.data?.nodes.some((n) => n.hardwareProbing) ? 3000 : 10000),
  })

/** Who the panel is in the cluster: endpoint, credentials and permissions. */
export const useClusterIdentity = () =>
  useQuery({ queryKey: keys.clusterIdentity, queryFn: api.cluster.getIdentity, staleTime: 60_000 })

/** Dependency checks for the sidebar cluster card (admins only). */
export const useClusterHealth = () =>
  useQuery({ queryKey: keys.clusterHealth, queryFn: api.cluster.getHealth, refetchInterval: 60_000 })

/** Starts the hardware probes of all nodes. */
export const useProbeNodes = (cb?: MutationCallbacks<NodesResponse, void>) =>
  useApiMutation(api.cluster.probeNodes, (qc, data) => qc.setQueryData(keys.clusterNodes, data), cb)
