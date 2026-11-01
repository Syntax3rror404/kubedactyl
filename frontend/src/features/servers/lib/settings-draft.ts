import { serverName } from "@/lib/format"
import type { GameServer, UpdateServerRequest } from "@/lib/types"

/** Pool selection of a server without load balancer (Cilium pool names have no capitals). */
export const NO_LOAD_BALANCER = "ClusterIP"

/** The pool selection of a saved server: its pool, or NO_LOAD_BALANCER. */
export function poolValue(server: GameServer) {
  return server.spec.serviceType === "ClusterIP" ? NO_LOAD_BALANCER : (server.spec.loadBalancerPool ?? "")
}

/** The service type and pool of a pool selection. */
export function poolRequest(pool: string) {
  return pool === NO_LOAD_BALANCER
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
    sharedWith: s.sharedWith ?? [],
  }
}

export type SettingsDraft = ReturnType<typeof settingsDraft>

/** Change handler of the settings cards. */
type SettingsChange = (patch: Partial<SettingsDraft>) => void

/** What every card of the settings page gets: the saved server, the draft and the errors of the last save. */
export interface SettingsCardProps {
  server: GameServer
  draft: SettingsDraft
  onChange: SettingsChange
  errors: Record<string, string>
}

// The request fields of each draft field.
const requestOf: { [K in keyof SettingsDraft]: (value: SettingsDraft[K]) => UpdateServerRequest } = {
  displayName: (displayName) => ({ displayName }),
  memory: (memoryMiB) => ({ memoryMiB }),
  cpu: (cpuMillis) => ({ cpuMillis }),
  disk: (diskMiB) => ({ diskMiB }),
  ports: (ports) => ({ ports }),
  lbIP: (ip) => ({ loadBalancerIP: ip.trim() }),
  trafficPolicy: (externalTrafficPolicy) => ({ externalTrafficPolicy }),
  ipv6: (ipv6) => ({ ipv6 }),
  pool: poolRequest,
  crashRestart: (crashRestart) => ({ crashRestart }),
  stopTimeout: (stopTimeoutSeconds) => ({ stopTimeoutSeconds }),
  sharedWith: (sharedWith) => ({ sharedWith }),
}

// What users may change; the rest is for administrators.
const userFields = new Set<keyof SettingsDraft>(["displayName", "crashRestart", "pool", "sharedWith"])

/**
 * The update request: only the fields that differ from the saved draft, so a save never reverts what someone else
 * changed meanwhile. Users may only change the name, crash restarts, the pool and the subusers who see it.
 */
export function settingsChanges(saved: SettingsDraft, d: SettingsDraft, isAdmin: boolean): UpdateServerRequest {
  const changed = (Object.keys(requestOf) as (keyof SettingsDraft)[]).filter(
    (k) => (isAdmin || userFields.has(k)) && JSON.stringify(d[k]) !== JSON.stringify(saved[k]),
  )
  return Object.assign({}, ...changed.map((k) => (requestOf[k] as (v: unknown) => UpdateServerRequest)(d[k])))
}
