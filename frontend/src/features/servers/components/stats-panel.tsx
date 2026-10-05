import { ClockIcon, CpuIcon, HardDriveIcon, MemoryStickIcon } from "lucide-react"

import { UsageBar } from "@/components/common/usage-bar"
import { ChartStatTile, StatTile } from "@/features/servers/components/stat-tile"
import { useSamples } from "@/features/servers/hooks/use-samples"
import { activePhases, formatBytes, formatCpuPercent, formatDuration, formatMiB, phaseOf } from "@/lib/format"
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
