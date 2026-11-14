// Formatting helpers every app uses: sizes, CPU, durations, relative times, counts, names, roles and the titles of
// the legal texts.

import { roles } from "@/app/config"
import type { LegalTexts, Role, UserView } from "@/base/lib/types"

/** A size in binary units ("1.5 GiB"); "-" while unknown. */
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

/** A duration in its two largest units ("2h 5m"); "-" for none. */
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

/** How long ago ("5 min ago"), the date after a day; "-" for none. */
export function formatRelativeTime(iso?: string): string {
  if (!iso) return "-"
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  if (diff < 60) return "just now"
  if (diff < 3600) return `${Math.floor(diff / 60)} min ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)} h ago`
  return formatDate(iso)
}

/** A count with its word, plural unless 1 ("2 files", "1 file"). */
export function plural(n: number, word: string): string {
  return `${n} ${word}${n === 1 ? "" : "s"}`
}

type Named = Pick<UserView, "username" | "displayName">

/** The display name of a user, else the username. */
export function userName(u: Named): string {
  return u.displayName || u.username
}

/** The role as the UI names it (app/config.ts). */
export function roleLabel(role: Role): string {
  return roles.find((r) => r.value === role)?.label ?? role
}

/** Two letters for avatars ("Alice Smith" -> "AS", "bob" -> "BO"). */
export function initials(u: Named) {
  const source = userName(u).trim()
  const parts = source.split(/\s+/)
  return (parts.length > 1 ? parts[0][0] + parts[1][0] : source.slice(0, 2)).toUpperCase()
}

/** CPU in millicores as cores ("1.5 cores"); "-" while unknown. */
export function formatCores(millis: number | null | undefined): string {
  if (millis == null) return "-"
  const cores = millis / 1000
  return `${cores >= 10 || Number.isInteger(cores) ? cores.toFixed(0) : cores.toFixed(1)} ${cores === 1 ? "core" : "cores"}`
}

/** value as percent of max; "-" while either is unknown. */
export function formatPercent(value: number | null | undefined, max: number | null | undefined): string {
  if (value == null || !max) return "-"
  return `${Math.round((value / max) * 100)}%`
}

/** A legal text an administrator can publish (settings, footer of every page). */
export type LegalDoc = keyof LegalTexts

/** Title of each legal text, in the order of the footer links. */
export const legalTitles: Record<LegalDoc, string> = {
  legalNotice: "Legal notice",
  privacyPolicy: "Privacy policy",
}

/** The legal texts in the order of the footer links. */
export const legalDocs = Object.keys(legalTitles) as LegalDoc[]
