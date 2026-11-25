import { activePhases, matches, phaseOf, serverEggName, serverName, serverOwner } from "@/lib/format"
import type { Egg, GameServer } from "@/lib/types"

/** Which servers the status buttons show: all, those with a game process (running) or the others (stopped). */
export type StatusFilter = "all" | "running" | "stopped"

/** The search and the status button of the dashboard. */
export interface ServerFilterState {
  query: string
  status: StatusFilter
}

/** Whether the server counts as running for the tile and the filter: its game process exists. */
export const isActive = (gs: GameServer) => activePhases.includes(phaseOf(gs))

/** Applies the filter: every word of the query occurs in the display name, server name, egg or owner. */
export function filterServers(list: GameServer[], eggs: Map<string, Egg>, f: ServerFilterState): GameServer[] {
  return list.filter((s) => {
    const active = isActive(s)
    if ((f.status === "running" && !active) || (f.status === "stopped" && active)) return false
    return matches(f.query, serverName(s), s.metadata.name, serverEggName(s, eggs.get(s.spec.eggRef)), serverOwner(s))
  })
}
