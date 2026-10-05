// Search and status filter of the dashboard server list.

import { activePhases, phaseOf, serverEggName, serverName, serverOwner } from "@/lib/format"
import type { Egg, GameServer } from "@/lib/types"

export type StatusFilter = "all" | "running" | "stopped"

export interface ServerFilterState {
  query: string
  status: StatusFilter
}

/** Applies the filter: the query matches display name, server name, egg and owner. */
export function filterServers(list: GameServer[], eggs: Map<string, Egg>, f: ServerFilterState): GameServer[] {
  const q = f.query.trim().toLowerCase()
  return list.filter((s) => {
    const active = activePhases.includes(phaseOf(s))
    if ((f.status === "running" && !active) || (f.status === "stopped" && active)) return false
    if (!q) return true
    const haystack = [serverName(s), s.metadata.name, serverEggName(s, eggs.get(s.spec.eggRef)), serverOwner(s) ?? ""]
      .join(" ")
      .toLowerCase()
    return haystack.includes(q)
  })
}
