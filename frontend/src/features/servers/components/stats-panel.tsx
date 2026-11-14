import { lazy, Suspense } from "react"
import { ClockIcon, CpuIcon, HardDriveIcon, MemoryStickIcon } from "lucide-react"

import { UsageBar } from "@/base/components/usage-bar"
import { formatBytes, formatDuration } from "@/base/lib/format"
import { type Sample, useSamples } from "@/features/servers/hooks/use-samples"
import { activePhases, formatCpuPercent, formatMiB, phaseOf } from "@/lib/format"
import { useServerStats } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Live CPU, memory, disk and uptime of a server. */
export function StatsPanel({ server }: { server: GameServer }) {
  const stats = useServerStats(server.metadata.name)
  const samples = useSamples(stats.data, stats.dataUpdatedAt)
  const active = activePhases.includes(phaseOf(server))
  const r = server.spec.resources
  const live = (v: number | null | undefined, fmt: (v: number) => string) =>
    !active ? "Offline" : v == null ? "Collecting…" : fmt(v)

  return (
    <div className="grid content-start gap-4 sm:grid-cols-2 xl:grid-cols-1">
      <ChartStatTile
        icon={<CpuIcon />}
        label="CPU"
        value={live(stats.data?.cpuMillis, formatCpuPercent)}
        limit={r.cpuMillis ? `of ${formatCpuPercent(r.cpuMillis)}` : "unlimited"}
        data={samples}
        dataKey="cpu"
        color="var(--chart-cpu)"
        max={r.cpuMillis || undefined}
        active={active}
      />
      <ChartStatTile
        icon={<MemoryStickIcon />}
        label="Memory"
        value={live(stats.data?.memoryBytes, formatBytes)}
        limit={`of ${formatMiB(r.memoryMiB)}`}
        data={samples}
        dataKey="mem"
        color="var(--chart-mem)"
        max={r.memoryMiB * 1024 * 1024}
        active={active}
      />
      <StatTile
        icon={<HardDriveIcon />}
        label="Disk"
        value={formatBytes(stats.data?.diskBytes)}
        limit={`of ${formatMiB(r.diskMiB)}`}
      >
        <UsageBar value={stats.data?.diskBytes} max={r.diskMiB * 1024 * 1024} className="mt-3" />
      </StatTile>
      <StatTile
        icon={<ClockIcon />}
        label="Uptime"
        value={active ? formatDuration(stats.data?.uptimeSeconds ?? 0) : "-"}
        limit={server.status?.lastExitCode != null ? `last exit code ${server.status.lastExitCode}` : ""}
      />
    </div>
  )
}

// The chart library is large: loaded on its own, the console does not wait for it.
const StatChart = lazy(() => import("@/features/servers/components/stat-chart").then((m) => ({ default: m.StatChart })))

// Shown while there is nothing to draw (server offline, fewer than two samples) and while the chart loads.
const noChart = (
  <div className="mt-3 flex h-16 items-end">
    <div className="w-full border-t border-dashed" />
  </div>
)

/** One value of the stats panel (CPU, memory, disk, uptime). */
function StatTile({
  icon,
  label,
  value,
  limit,
  children,
}: {
  icon: React.ReactNode
  label: string
  value: string
  limit?: string
  children?: React.ReactNode
}) {
  return (
    <div className="rounded-2xl border bg-card p-4 shadow-sm">
      <div className="flex items-center gap-2 text-xs font-medium tracking-wide text-muted-foreground uppercase [&_svg]:size-3.5">
        {icon}
        {label}
      </div>
      <div className="mt-2 flex items-baseline gap-2">
        <span className="text-2xl font-semibold tabular-nums">{value}</span>
        {limit && <span className="text-xs text-muted-foreground">{limit}</span>}
      </div>
      {children}
    </div>
  )
}

/** Stat tile with a small area chart of the recent samples. */
function ChartStatTile({
  icon,
  label,
  value,
  limit,
  data,
  dataKey,
  color,
  max,
  active,
}: {
  icon: React.ReactNode
  label: string
  value: string
  limit: string
  data: Sample[]
  dataKey: "cpu" | "mem"
  color: string
  max?: number
  active: boolean
}) {
  return (
    <StatTile icon={icon} label={label} value={value} limit={limit}>
      {!active || data.length < 2 ? (
        noChart
      ) : (
        <Suspense fallback={noChart}>
          <StatChart label={label} data={data} dataKey={dataKey} color={color} max={max} />
        </Suspense>
      )}
    </StatTile>
  )
}
