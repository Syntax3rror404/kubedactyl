// Formatting helpers for the UI (sizes, CPU, durations, relative times, names, roles and phases).

import type { Egg, GameServer, Phase, Role, UserView } from "@/lib/types"

export function formatBytes(bytes: number | null | undefined, digits = 1): string {
  if (bytes == null) return "-"
  const units = ["B", "KiB", "MiB", "GiB", "TiB"]
  let v = bytes
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : digits)} ${units[i]}`
}

export function formatMiB(mib: number): string {
  return mib >= 1024 ? `${+(mib / 1024).toFixed(1)} GiB` : `${mib} MiB`
}

export function formatCpuPercent(millis: number | null | undefined): string {
  if (millis == null) return "-"
  return `${(millis / 10).toFixed(millis < 100 ? 1 : 0)}%`
}

export function formatDuration(seconds: number): string {
  if (seconds <= 0) return "-"
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

/** Seconds since the given time (0 for future times). */
export function secondsSince(iso: string): number {
  return Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
}

/** A date in the browser's format, e.g. 04/02/2024. */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString()
}

export function formatRelativeTime(iso?: string): string {
  if (!iso) return "-"
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  if (diff < 60) return "just now"
  if (diff < 3600) return `${Math.floor(diff / 60)} min ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)} h ago`
  return formatDate(iso)
}

export function serverName(gs: GameServer): string {
  return gs.spec.displayName || gs.metadata.name
}

/** Name of a server's egg; the egg reference while the egg is not loaded (or was deleted). */
export function serverEggName(gs: GameServer, egg?: Egg): string {
  return egg?.spec.displayName ?? gs.spec.eggRef
}

/** Address players connect to: the external domain (when configured) or the load balancer IP. */
export function serverAddress(gs: GameServer, externalDomain?: string): string | null {
  const ip = gs.status?.address
  if (!ip) return null
  return `${externalDomain || ip}:${gs.spec.ports[0]}`
}

export function phaseOf(gs?: GameServer): Phase {
  return gs?.status?.phase ?? "Pending"
}

/** A deleted server stays until its finalizer has released the volume; nothing can be done with it anymore. */
export function isRemoving(gs: GameServer): boolean {
  return !!gs.metadata.deletionTimestamp
}

/** Phases in which the game process exists. */
export const activePhases: Phase[] = ["Starting", "Running", "Stopping"]

/** Phases in which the server can be started (no game process, files can be restored). */
export const stoppedPhases: Phase[] = ["Offline", "InstallFailed"]

export function plural(n: number, word: string): string {
  return `${n} ${word}${n === 1 ? "" : "s"}`
}

type Named = Pick<UserView, "username" | "displayName">

export function userName(u: Named): string {
  return u.displayName || u.username
}

export function roleLabel(role: Role): string {
  return role === "admin" ? "Administrator" : "User"
}

/** Two letters for avatars ("Alice Smith" -> "AS", "bob" -> "BO"). */
export function initials(u: Named) {
  const source = userName(u).trim()
  const parts = source.split(/\s+/)
  return (parts.length > 1 ? parts[0][0] + parts[1][0] : source.slice(0, 2)).toUpperCase()
}

/** Username that owns a server (label kubedactyl.io/owner), if any. */
export function serverOwner(gs: GameServer): string | undefined {
  return gs.metadata.labels?.["kubedactyl.io/owner"]
}

export function formatCores(millis: number | null | undefined): string {
  if (millis == null) return "-"
  const cores = millis / 1000
  return `${cores >= 10 || Number.isInteger(cores) ? cores.toFixed(0) : cores.toFixed(1)} ${cores === 1 ? "core" : "cores"}`
}

/** CPU limit of a server (1000 = one core); 0 means no limit. */
export function formatCpuLimit(millis: number | undefined): string {
  return millis ? plural(millis / 1000, "core") : "Unlimited"
}

export function formatPercent(value: number | null | undefined, max: number | null | undefined): string {
  if (value == null || !max) return "-"
  return `${Math.round((value / max) * 100)}%`
}
