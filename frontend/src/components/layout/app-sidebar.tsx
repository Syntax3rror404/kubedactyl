import {
  ArrowUpCircleIcon,
  BookOpenIcon,
  EggIcon,
  LayoutGridIcon,
  NetworkIcon,
  PlusIcon,
  ServerIcon,
  Settings2Icon,
  UsersIcon,
} from "lucide-react"
import { Link, NavLink, useLocation } from "react-router"

import { BrandMark } from "@/components/common/brand-mark"
import { StatusDot } from "@/components/common/status-badge"
import { ClusterCard } from "@/components/layout/cluster-card"
import { UserMenu } from "@/components/layout/user-menu"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useAuth } from "@/hooks/use-auth"
import { phaseOf, serverName } from "@/lib/format"
import { useBranding, useServers, useSettings, useUpgradeStatus } from "@/lib/queries"

// Entries only administrators see; Settings carries the badge of an available update.
const adminNav = [
  { to: "/eggs", label: "Eggs", icon: EggIcon },
  { to: "/users", label: "Users", icon: UsersIcon },
  { to: "/cluster", label: "Cluster", icon: NetworkIcon },
  { to: "/settings", label: "Settings", icon: Settings2Icon },
]

/** Navigation (admin entries only for admins), the server list, the cluster card and the user menu. */
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation()
  const servers = useServers()
  const branding = useBranding().data
  const settings = useSettings()
  const { isAdmin } = useAuth()
  const upgrade = useUpgradeStatus(isAdmin)
  const updateAvailable = isAdmin && !!upgrade.data?.enabled && upgrade.data.newer.length > 0

  return (
    <Sidebar collapsible="icon" variant="inset" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <Link to="/">
                <BrandMark className="aspect-square size-8" iconClassName="size-4" />
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">{branding?.name}</span>
                  <span className="truncate text-xs text-muted-foreground">{branding?.tagline}</span>
                </div>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Platform</SidebarGroupLabel>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild tooltip="Dashboard" isActive={pathname === "/"}>
                <NavLink to="/">
                  <LayoutGridIcon />
                  <span>Dashboard</span>
                </NavLink>
              </SidebarMenuButton>
            </SidebarMenuItem>
            {isAdmin &&
              adminNav.map((item) => {
                const badge = item.to === "/settings" && updateAvailable
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      asChild
                      tooltip={badge ? `${item.label} · version ${upgrade.data?.newer[0]} available` : item.label}
                      isActive={pathname.startsWith(item.to)}
                    >
                      <NavLink to={item.to}>
                        <item.icon />
                        <span>{item.label}</span>
                      </NavLink>
                    </SidebarMenuButton>
                    {badge && (
                      <>
                        <SidebarMenuBadge className="gap-1 rounded-full bg-emerald-500/15 px-1.5 text-emerald-700! ring-1 ring-emerald-500/30 dark:text-emerald-300!">
                          <ArrowUpCircleIcon className="size-3" />
                          Update
                          <span className="sr-only">: version {upgrade.data?.newer[0]} is available</span>
                        </SidebarMenuBadge>
                        {/* Collapsed sidebar: the badge is hidden, a dot on the icon remains. */}
                        <span className="pointer-events-none absolute top-1.5 left-5 hidden size-2 rounded-full bg-emerald-500 ring-2 ring-sidebar group-data-[collapsible=icon]:block" />
                      </>
                    )}
                  </SidebarMenuItem>
                )
              })}
            {settings.data && !settings.data.disableApiDocs && (
              <SidebarMenuItem>
                <SidebarMenuButton asChild tooltip="API documentation">
                  <a href="/swagger/index.html" target="_blank" rel="noreferrer">
                    <BookOpenIcon />
                    <span>API docs</span>
                  </a>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )}
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Servers</SidebarGroupLabel>
          {isAdmin && (
            <SidebarGroupAction asChild title="New server">
              <Link to="/servers/new">
                <PlusIcon />
                <span className="sr-only">New server</span>
              </Link>
            </SidebarGroupAction>
          )}
          <SidebarMenu>
            {servers.isLoading &&
              Array.from({ length: 3 }).map((_, i) => (
                <SidebarMenuItem key={i}>
                  <SidebarMenuSkeleton showIcon />
                </SidebarMenuItem>
              ))}
            {servers.data?.map((gs) => {
              const to = `/servers/${gs.metadata.name}`
              return (
                <SidebarMenuItem key={gs.metadata.uid}>
                  <SidebarMenuButton asChild tooltip={serverName(gs)} isActive={pathname.startsWith(to)}>
                    <NavLink to={to}>
                      <ServerIcon />
                      <span>{serverName(gs)}</span>
                    </NavLink>
                  </SidebarMenuButton>
                  <SidebarMenuBadge>
                    <StatusDot phase={phaseOf(gs)} />
                  </SidebarMenuBadge>
                </SidebarMenuItem>
              )
            })}
            {servers.data?.length === 0 && (
              <p className="px-2 py-1 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
                No servers yet.
              </p>
            )}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        {isAdmin && <ClusterCard />}
        <UserMenu />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
