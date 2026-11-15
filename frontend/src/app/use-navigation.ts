// What this app shows in the shared navigation (components/navigation): the panel name with the cluster state for
// administrators, the servers as the center, the platform pages and the administration with the API documentation
// above the user menu.

import { useLocation } from "react-router"
import {
  ArrowUpCircleIcon,
  BookOpenIcon,
  EggIcon,
  LayoutGridIcon,
  NetworkIcon,
  Settings2Icon,
  UsersIcon,
} from "lucide-react"

import { useAuth } from "@/base/hooks/use-auth"
import { plural } from "@/base/lib/format"
import { useBranding, useSettings, useUpgradeStatus } from "@/base/lib/queries"
import type { NavConfig, NavGroup, NavItem, NavTone } from "@/base/types"
import { useServersNav } from "@/features/servers/hooks/use-servers-nav"
import { useClusterHealth, useClusterInfo } from "@/lib/queries"
import type { CheckStatus } from "@/lib/types"

const healthTone: Record<CheckStatus, NavTone> = {
  ok: "success",
  warning: "warning",
  error: "danger",
  skipped: "muted",
  info: "info",
}

/** The header: name of the panel; for administrators the cluster checks as short diagnosis and the version. */
function usePanelHeader(isAdmin: boolean): NavConfig["header"] {
  const branding = useBranding().data
  const cluster = useClusterInfo(isAdmin).data
  const health = useClusterHealth(isAdmin).data
  const name = branding?.name ?? ""
  if (!isAdmin || !health) return { name, detail: branding?.tagline }
  const checks = health.checks.map((c) => ({ label: c.label, message: c.message, tone: healthTone[c.status] }))
  const problems = checks.filter((c) => c.tone === "warning" || c.tone === "danger").length
  const state = problems ? plural(problems, "problem") : "Healthy"
  return {
    name,
    detail: cluster ? `${state} · ${cluster.version}` : state,
    tone: healthTone[health.status],
    checks,
    checksLink: { label: "Cluster details", to: "/cluster" },
  }
}

/**
 * The group above the user menu: users and settings for administrators (Settings carries the update notice), and the
 * API documentation for everyone unless switched off.
 */
function useFooter(at: (to: string) => boolean, isAdmin: boolean): NavGroup | undefined {
  const upgrade = useUpgradeStatus(isAdmin).data
  const settings = useSettings().data
  const newer = upgrade?.enabled ? upgrade.newer[0] : undefined
  const items: NavItem[] = []
  if (isAdmin) {
    items.push(
      { label: "Users", to: "/users", icon: UsersIcon, active: at("/users") },
      {
        label: "Settings",
        to: "/settings",
        icon: Settings2Icon,
        active: at("/settings"),
        badge: newer
          ? { kind: "status", tone: "success", text: `Update ${newer}`, icon: ArrowUpCircleIcon }
          : undefined,
      },
    )
  }
  if (settings && !settings.disableApiDocs) {
    items.push({ label: "API docs", href: "/api-docs", icon: BookOpenIcon })
  }
  if (items.length === 0) return undefined
  return { label: isAdmin ? "Administration" : undefined, items }
}

/** The navigation of this app, for components/navigation/app-sidebar.tsx. */
export function useNavigation(): NavConfig {
  const pathname = useLocation().pathname.replace(/(.)\/+$/, "$1")
  const { user, isAdmin } = useAuth()
  const at = (to: string) => pathname === to || pathname.startsWith(`${to}/`)
  const header = usePanelHeader(isAdmin)
  const servers = useServersNav(pathname, isAdmin)
  const footer = useFooter(at, isAdmin)
  const platform: NavItem[] = [
    { label: "Eggs", to: "/eggs", icon: EggIcon, active: at("/eggs") },
    { label: "Cluster", to: "/cluster", icon: NetworkIcon, active: at("/cluster") },
  ]
  return {
    header,
    groups: [
      { items: [{ label: "Dashboard", to: "/", icon: LayoutGridIcon, active: pathname === "/" }, servers] },
      ...(isAdmin ? [{ label: "Platform", items: platform }] : []),
    ],
    footer,
    userDetail: user.namespace,
    search: [],
  }
}
