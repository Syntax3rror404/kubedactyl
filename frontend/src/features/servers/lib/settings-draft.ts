import { serverName } from "@/lib/format"
import type { GameServer, UpdateServerRequest } from "@/lib/types"

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
    pool: s.loadBalancerPool ?? "",
    crashRestart: s.crashRestart ?? true,
    stopTimeout: s.stopTimeoutSeconds ?? 600,
  }
}

export type SettingsDraft = ReturnType<typeof settingsDraft>

/** Change handler of the settings cards. */
export type SettingsChange = (patch: Partial<SettingsDraft>) => void

/** The update request; users may only change the name, crash restarts and the pool. */
export function settingsChanges(server: GameServer, d: SettingsDraft, isAdmin: boolean): UpdateServerRequest {
  const pool = d.pool !== (server.spec.loadBalancerPool ?? "") && { loadBalancerPool: d.pool }
  if (!isAdmin) return { displayName: d.displayName, crashRestart: d.crashRestart, ...pool }
  return {
    displayName: d.displayName,
    memoryMiB: d.memory,
    cpuMillis: d.cpu,
    diskMiB: d.disk,
    ports: d.ports,
    loadBalancerIP: d.lbIP.trim(),
    externalTrafficPolicy: d.trafficPolicy,
    crashRestart: d.crashRestart,
    stopTimeoutSeconds: d.stopTimeout,
    ...pool,
  }
}
