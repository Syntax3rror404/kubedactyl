import { PlusIcon, ServerIcon } from "lucide-react"

import type { NavItem, NavTone } from "@/base/types"
import { phaseOf, phaseStatus, serverName } from "@/lib/format"
import { useServers } from "@/lib/queries"

/**
 * The center of the sidebar: the servers with their phase, running of all as the branch's badge (green while one
 * runs), and "New server" for administrators. pathname marks the open server.
 */
export function useServersNav(pathname: string, isAdmin: boolean): NavItem {
  const servers = useServers().data ?? []
  const phases = servers.map(phaseOf)
  const running = phases.filter((p) => p === "Running").length
  const branchTone: NavTone = phases.includes("InstallFailed") ? "danger" : running ? "success" : "muted"
  const children: NavItem[] = servers.map((gs, i) => {
    const to = `/servers/${gs.metadata.name}`
    const { tone, pulse } = phaseStatus[phases[i]] ?? phaseStatus.Pending
    return {
      label: serverName(gs),
      to,
      icon: ServerIcon,
      active: pathname === to || pathname.startsWith(`${to}/`),
      badge: { kind: "status", tone, pulse },
    }
  })
  if (isAdmin) {
    children.push({ label: "New server", to: "/servers/new", icon: PlusIcon, active: pathname === "/servers/new" })
  }
  return {
    label: "Servers",
    icon: ServerIcon,
    children,
    open: true,
    empty: "No servers yet.",
    badge: servers.length ? { kind: "status", text: `${running}/${servers.length}`, tone: branchTone } : undefined,
  }
}
