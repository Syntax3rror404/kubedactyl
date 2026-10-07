import {
  AlertTriangleIcon,
  ArchiveIcon,
  CalendarClockIcon,
  DownloadCloudIcon,
  FolderIcon,
  SettingsIcon,
  SlidersHorizontalIcon,
  StethoscopeIcon,
  TerminalIcon,
  ZapIcon,
} from "lucide-react"
import { Link, Outlet, useLocation, useParams } from "react-router"

import { Callout } from "@/components/common/callout"
import { CopyButton } from "@/components/common/copy-button"
import { EggIcon } from "@/components/common/egg-icon"
import { QueryState } from "@/components/common/query-state"
import { ScrollableTabsList } from "@/components/common/scrollable-tabs-list"
import { RemovingBadge, StatusBadge, SuspendedBadge } from "@/components/common/status-badge"
import { PageHeader } from "@/components/layout/page-header"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsTrigger } from "@/components/ui/tabs"
import { FilesPodLight } from "@/features/servers/components/files-pod-light"
import { PowerControls } from "@/features/servers/components/power-controls"
import { RestartBanner } from "@/features/servers/components/restart-banner"
import { ServerNoticeDialog } from "@/features/servers/components/server-notice-dialog"
import { RemovingNotice, SuspendedBanner, SuspendedNotice } from "@/features/servers/components/locked-notice"
import { useAuth } from "@/hooks/use-auth"
import { isRemoving, phaseOf, serverAddress, serverEggName, serverName } from "@/lib/format"
import { useEgg, useServer, useSettings } from "@/lib/queries"
import type { Egg, GameServer } from "@/lib/types"

export interface ServerContext {
  server: GameServer
  egg?: Egg
}

const tabs = [
  { to: "", label: "Console", icon: TerminalIcon },
  { to: "files", label: "Files", icon: FolderIcon },
  { to: "startup", label: "Startup", icon: SlidersHorizontalIcon },
  { to: "schedules", label: "Schedules", icon: CalendarClockIcon },
  { to: "tasks", label: "Tasks", icon: ZapIcon },
  { to: "backups", label: "Backups", icon: ArchiveIcon },
  { to: "diagnostics", label: "Diagnostics", icon: StethoscopeIcon },
  { to: "settings", label: "Settings", icon: SettingsIcon },
]

/** Frame of all server pages: header with status and power buttons, tabs, notices; locked view for suspended servers
 * (owners) and servers being removed (everyone). */
export function ServerLayout() {
  const { server: name = "" } = useParams()
  const server = useServer(name)
  return (
    <QueryState
      query={server}
      skeleton={
        <div className="space-y-6">
          <Skeleton className="h-20 rounded-2xl" />
          <Skeleton className="h-[60vh] rounded-2xl" />
        </div>
      }
    >
      {(gs) => <ServerFrame gs={gs} />}
    </QueryState>
  )
}

function ServerFrame({ gs }: { gs: GameServer }) {
  const name = gs.metadata.name
  const { pathname } = useLocation()
  const egg = useEgg(gs.spec.eggRef)
  const settings = useSettings()
  const domain = settings.data?.externalDomain
  const { isAdmin } = useAuth()
  const phase = phaseOf(gs)
  const address = serverAddress(gs, domain)
  const base = `/servers/${name}`
  const current = pathname.slice(base.length).replace(/^\//, "").split("/")[0]
  // Owners of a suspended server only see the notice; admins keep full access. A server being removed is locked
  // for everyone.
  const removing = isRemoving(gs)
  const suspended = !!gs.spec.suspended
  const locked = removing || (suspended && !isAdmin)
  const eggName = serverEggName(gs, egg.data)
  const installer = egg.data?.spec.install?.container

  return (
    <div className="space-y-6">
      <PageHeader
        icon={<EggIcon name={eggName} icon={egg.data?.spec.icon} className="size-14 rounded-xl" />}
        title={serverName(gs)}
        badges={
          <>
            {removing ? <RemovingBadge /> : <StatusBadge phase={phase} />}
            {suspended && <SuspendedBadge />}
          </>
        }
        description={
          <span className="flex flex-wrap items-center gap-2">
            <Link to={`/eggs/${gs.spec.eggRef}`} className="hover:text-foreground">
              {eggName}
            </Link>
            <span>·</span>
            {address ? (
              <span className="inline-flex items-center font-mono text-foreground">
                {address}
                <CopyButton value={address} label="Copy address" />
              </span>
            ) : (
              <span>waiting for load balancer IP…</span>
            )}
          </span>
        }
        actions={(removing || !locked) && <PowerControls server={gs} />}
      />

      {/* Users see the administrator's notice every time they open a server. */}
      <ServerNoticeDialog server={name} notice={isAdmin ? undefined : settings.data?.serverNotice} />
      {!locked && <RestartBanner server={gs} />}
      {suspended && !locked && <SuspendedBanner server={gs} />}

      {phase === "Installing" && !locked && (
        <Callout
          tone="info"
          icon={<DownloadCloudIcon />}
          title={`Running the install script${installer ? ` in ${installer}` : ""}…`}
          progress
        >
          <p className="text-muted-foreground">Follow the output in the console. {gs.status?.message}</p>
        </Callout>
      )}
      {gs.status?.message && phase !== "Installing" && !suspended && !removing && (
        <Callout tone="warning" icon={<AlertTriangleIcon />}>
          <p className="break-all">{gs.status.message}</p>
        </Callout>
      )}

      {removing ? (
        <RemovingNotice />
      ) : locked ? (
        <SuspendedNotice />
      ) : (
        <>
          <Tabs value={current}>
            <ScrollableTabsList>
              {tabs.map((t) => (
                <TabsTrigger key={t.to} value={t.to} asChild>
                  <Link to={t.to ? `${base}/${t.to}` : base}>
                    <t.icon />
                    {t.label}
                    {t.to === "files" && <FilesPodLight server={name} />}
                  </Link>
                </TabsTrigger>
              ))}
            </ScrollableTabsList>
          </Tabs>

          {/* Keyed by server so form state of the tabs starts fresh for every server */}
          <Outlet key={name} context={{ server: gs, egg: egg.data } satisfies ServerContext} />
        </>
      )}
    </div>
  )
}
