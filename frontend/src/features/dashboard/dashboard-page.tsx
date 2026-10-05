import { useState } from "react"
import {
  HardDriveIcon,
  EggIcon as EggLucide,
  MemoryStickIcon,
  PlusIcon,
  ServerIcon,
  UploadIcon,
  ZapIcon,
} from "lucide-react"
import { Link } from "react-router"

import { MetricTile } from "@/components/common/metric-tile"
import { EmptyState, QueryState, SkeletonGrid } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { ServerCard } from "@/features/dashboard/components/server-card"
import { ServerFilter } from "@/features/dashboard/components/server-filter"
import { filterServers, type ServerFilterState } from "@/features/dashboard/lib/server-filter"
import { useAuth } from "@/hooks/use-auth"
import { activePhases, formatMiB, phaseOf, userName } from "@/lib/format"
import { useClusterInfo, useEggs, useServers } from "@/lib/queries"

/** /: the servers the user may see as cards with live usage; search and status filter from 7 servers on. */
export function DashboardPage() {
  const servers = useServers()
  const eggs = useEggs()
  const { isAdmin, user } = useAuth()
  const cluster = useClusterInfo(isAdmin)
  const list = servers.data ?? []
  const running = list.filter((s) => phaseOf(s) === "Running").length
  const eggByName = new Map((eggs.data ?? []).map((e) => [e.metadata.name, e]))
  const memory = list.reduce((sum, s) => sum + s.spec.resources.memoryMiB, 0)
  // Search and status filter appear once there are more servers than fit on one screen.
  const [filter, setFilter] = useState<ServerFilterState>({ query: "", status: "all" })
  const showFilter = list.length > 6
  const shown = showFilter ? filterServers(list, eggByName, filter) : list
  const active = list.filter((s) => activePhases.includes(phaseOf(s))).length

  return (
    <div className="space-y-8">
      <PageHeader
        title={isAdmin ? "Game servers" : `Welcome, ${userName(user)}`}
        description={isAdmin ? "Everything running on your cluster, live." : "Your game servers, live."}
        actions={
          isAdmin && (
            <>
              <Button variant="outline" asChild>
                <Link to="/eggs">
                  <UploadIcon />
                  Import egg
                </Link>
              </Button>
              <Button asChild>
                <Link to="/servers/new">
                  <PlusIcon />
                  New server
                </Link>
              </Button>
            </>
          )
        }
      />

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <MetricTile icon={<ServerIcon />} label="Servers" value={servers.isLoading ? null : String(list.length)} />
        <MetricTile
          icon={<ZapIcon />}
          label="Running"
          value={servers.isLoading ? null : `${running} / ${list.length}`}
          accent
        />
        <MetricTile
          icon={<MemoryStickIcon />}
          label="Memory allocated"
          value={servers.isLoading ? null : formatMiB(memory)}
        />
        {isAdmin ? (
          <MetricTile
            icon={<EggLucide />}
            label="Eggs"
            value={eggs.isLoading ? null : String(eggs.data?.length ?? 0)}
            hint={cluster.data?.version ? `k8s ${cluster.data.version}` : undefined}
          />
        ) : (
          <MetricTile
            icon={<HardDriveIcon />}
            label="Disk allocated"
            value={servers.isLoading ? null : formatMiB(list.reduce((sum, s) => sum + s.spec.resources.diskMiB, 0))}
          />
        )}
      </div>

      <QueryState
        query={servers}
        skeleton={<SkeletonGrid className="h-60" />}
        empty={<EmptyServers hasEggs={(eggs.data?.length ?? 0) > 0} isAdmin={isAdmin} />}
      >
        {() => (
          <div className="space-y-4">
            {showFilter && (
              <ServerFilter
                value={filter}
                onChange={setFilter}
                counts={{ all: list.length, running: active, stopped: list.length - active }}
              />
            )}
            {shown.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">No server matches the filter.</p>
            ) : (
              <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
                {shown.map((gs) => (
                  <ServerCard key={gs.metadata.uid} server={gs} egg={eggByName.get(gs.spec.eggRef)} />
                ))}
              </div>
            )}
          </div>
        )}
      </QueryState>
    </div>
  )
}

function EmptyServers({ hasEggs, isAdmin }: { hasEggs: boolean; isAdmin: boolean }) {
  if (!isAdmin) {
    return (
      <EmptyState
        icon={<ServerIcon />}
        title="No game servers yet"
        description="Your administrator has not assigned a server to you yet."
      />
    )
  }
  return (
    <EmptyState
      icon={<ServerIcon />}
      title="No game servers yet"
      description={
        hasEggs
          ? "Pick an egg and launch your first server in under a minute."
          : "Import a Pterodactyl or Pelican egg first, then create a server from it."
      }
    >
      <Button asChild>
        <Link to={hasEggs ? "/servers/new" : "/eggs"}>
          {hasEggs ? <PlusIcon /> : <UploadIcon />}
          {hasEggs ? "Create server" : "Import an egg"}
        </Link>
      </Button>
    </EmptyState>
  )
}
