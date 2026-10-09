import { serverName } from "@/lib/format"
import type { GameServer, UpdateServerRequest } from "@/lib/types"

/** Pool selection of a server without load balancer (Cilium pool names have no capitals). */
export const noLoadBalancer = "ClusterIP"

/** The pool selection of a saved server: its pool, or noLoadBalancer. */
export function poolValue(server: GameServer) {
  return server.spec.serviceType === "ClusterIP" ? noLoadBalancer : (server.spec.loadBalancerPool ?? "")
}

/** The service type and pool of a pool selection. */
export function poolRequest(pool: string) {
  return pool === noLoadBalancer
    ? ({ serviceType: "ClusterIP" } as const)
    : ({ serviceType: "LoadBalancer", loadBalancerPool: pool } as const)
}

/** Form values of the server settings page, starting from the saved server. */
export function settingsDraft(server: GameServer) {
  const s = server.spec
  return {
    displayName: serverName(server),
    memory: s.resources.memoryMiB,
    cpu: s.resources.cpuMillis ?? 0,
    disk: s.resources.diskMiB,
    ports: s.ports,
    lbIP: s.loadBalancerIP ?? "",
    trafficPolicy: s.externalTrafficPolicy ?? "Local",
    ipv6: s.ipv6 ?? false,
    pool: poolValue(server),
    crashRestart: s.crashRestart ?? true,
    stopTimeout: s.stopTimeoutSeconds ?? 600,
  }
}

export type SettingsDraft = ReturnType<typeof settingsDraft>

/** Change handler of the settings cards. */
export type SettingsChange = (patch: Partial<SettingsDraft>) => void

/** The update request; users may only change the name, crash restarts and the pool. */
export function settingsChanges(server: GameServer, d: SettingsDraft, isAdmin: boolean): UpdateServerRequest {
  const pool = d.pool !== poolValue(server) && poolRequest(d.pool)
  if (!isAdmin) return { displayName: d.displayName, crashRestart: d.crashRestart, ...pool }
  return {
    displayName: d.displayName,
    memoryMiB: d.memory,
    cpuMillis: d.cpu,
    diskMiB: d.disk,
    ports: d.ports,
    loadBalancerIP: d.lbIP.trim(),
    externalTrafficPolicy: d.trafficPolicy,
    ipv6: d.ipv6,
    crashRestart: d.crashRestart,
    stopTimeoutSeconds: d.stopTimeout,
    ...pool,
  }
}
