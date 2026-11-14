// Formatting helpers of this app (servers, addresses, pools, phases) and search matching; the shared ones are in
// base/lib/format.ts.

import { plural } from "@/base/lib/format"
import type { UserView } from "@/base/lib/types"
import type { NavTone } from "@/base/types"
import type { Egg, GameServer, Phase, Pool, ShareAccess } from "@/lib/types"

/** A size given in MiB, from 1024 on in GiB ("2 GiB"). */
export function formatMiB(mib: number): string {
  return mib >= 1024 ? `${+(mib / 1024).toFixed(1)} GiB` : `${mib} MiB`
}

/** CPU use in millicores as percent of one core ("150%"); "-" while unknown. */
export function formatCpuPercent(millis: number | null | undefined): string {
  if (millis == null) return "-"
  return `${(millis / 10).toFixed(millis < 100 ? 1 : 0)}%`
}

/** The name users gave the server, else its object name. */
export function serverName(gs: GameServer): string {
  return gs.spec.displayName || gs.metadata.name
}

/** Name of a server's egg; the egg reference while the egg is not loaded (or was deleted). */
export function serverEggName(gs: GameServer, egg?: Egg): string {
  return egg?.spec.displayName ?? gs.spec.eggRef
}

/** Whether an IP is an IPv6 address. */
export const isIPv6 = (ip: string) => ip.includes(":")

/** "host:port"; IPv6 addresses in brackets ("[2001:db8::5]:25565"). */
function hostPort(host: string, port: number): string {
  return isIPv6(host) ? `[${host}]:${port}` : `${host}:${port}`
}

/**
 * The load balancer IPs of a server (one per IP family, the main family first), or the DNS name of a
 * server without load balancer.
 */
export function serverIPs(gs: GameServer): string[] {
  return gs.status?.addresses?.length ? gs.status.addresses : [gs.status?.address ?? ""].filter(Boolean)
}

/**
 * Every address of a server: the external domain, or each load balancer IP (one per IP family). A server
 * without load balancer keeps its DNS name inside the cluster.
 */
export function serverAddresses(gs: GameServer, externalDomain?: string): string[] {
  const ips = serverIPs(gs)
  if (!ips.length) return []
  const domain = gs.spec.serviceType !== "ClusterIP" && externalDomain
  return (domain ? [domain] : ips).map((host) => hostPort(host, gs.spec.ports[0]))
}

/** Address players connect to: the external domain (when configured) or the load balancer IP. */
export function serverAddress(gs: GameServer, externalDomain?: string): string | null {
  return serverAddresses(gs, externalDomain)[0] ?? null
}

/** The phase of a server; "Pending" until the controller has seen it. */
export function phaseOf(gs?: GameServer): Phase {
  return gs?.status?.phase ?? "Pending"
}

/** The server files move to a volume of another storage class; the server is locked until they have. */
export function isMigrating(gs: GameServer): boolean {
  const { storageClass } = gs.spec
  const current = gs.status?.storageClass
  return !!storageClass && !!current && storageClass !== current
}

/** What the user may do on a server: everything (owner, admin), or what the owner shared with a subuser. */
export function serverAccess(gs: GameServer, user: UserView): "full" | ShareAccess {
  if (!user.owner) return "full"
  return gs.spec.sharedWith?.find((s) => s.user === user.username)?.access ?? "view"
}

/** A deleted server stays until its finalizer has released the volume; nothing can be done with it anymore. */
export function isRemoving(gs: GameServer): boolean {
  return !!gs.metadata.deletionTimestamp
}

/** Phases in which the game process exists. */
export const activePhases: Phase[] = ["Starting", "Running", "Stopping"]

/** Phases in which the server can be started (no game process, files can be restored). */
export const stoppedPhases: Phase[] = ["Offline", "InstallFailed"]

/** A whole number with thousands separators, as progress texts show file counts ("1,204"). */
export const formatNumber = (n: number) => Math.round(n).toLocaleString("en-US")

/** Large counts in short form (IPv6 pools hold 2^64 addresses: "1.8e19"). */
export function formatCount(n: number): string {
  return n < 1e9 ? `${n}` : n.toExponential(1).replace("e+", "e")
}

/** Size of an address block: a power of two for large ones ("2^64"), else the count. */
export function formatPoolSize(n: number): string {
  const bits = Math.log2(n)
  return n >= 2 ** 20 && Number.isInteger(bits) ? `2^${bits}` : formatCount(n)
}

/** Address usage of a pool per IP family; one row without family when only Cilium's total is known. */
export function poolUsage(p: Pool): { family?: string; total: number; available: number; used: number }[] {
  if (p.families?.length) return p.families
  return p.ipsTotal >= 0 ? [{ total: p.ipsTotal, available: p.ipsAvailable, used: p.ipsUsed }] : []
}

/** "IPv4 63 of 64 free"; IPv6 counts the used addresses ("IPv6 1 used of 2^64"): a block never runs out. */
export function formatPoolUsage(u: ReturnType<typeof poolUsage>[number]): string {
  const text =
    u.family === "IPv6"
      ? `${formatCount(u.used)} used of ${formatPoolSize(u.total)}`
      : `${formatCount(u.available)} of ${formatCount(u.total)} free`
  return u.family ? `${u.family} ${text}` : text
}

/** Whether every word of the search occurs in one of the texts (case-insensitive); search fields use it. */
export function matches(search: string, ...texts: (string | undefined)[]) {
  const text = texts.join(" ").toLowerCase()
  return search
    .toLowerCase()
    .split(/\s+/)
    .every((word) => text.includes(word))
}

/** Username that owns a server (label kubedactyl.io/owner), if any. */
export function serverOwner(gs: GameServer): string | undefined {
  return gs.metadata.labels?.["kubedactyl.io/owner"]
}

/** CPU limit of a server (1000 = one core); 0 means no limit. */
export function formatCpuLimit(millis: number | undefined): string {
  return millis ? plural(millis / 1000, "core") : "Unlimited"
}

/** Label and color of every server phase, for the badge and the sidebar; changing phases pulse. */
export const phaseStatus: Record<Phase, { label: string; tone: NavTone; pulse?: boolean }> = {
  Running: { label: "Running", tone: "success" },
  Starting: { label: "Starting", tone: "pending", pulse: true },
  Stopping: { label: "Stopping", tone: "pending", pulse: true },
  Installing: { label: "Installing", tone: "info", pulse: true },
  InstallFailed: { label: "Install failed", tone: "danger" },
  Offline: { label: "Offline", tone: "muted" },
  Pending: { label: "Pending", tone: "pending", pulse: true },
}
