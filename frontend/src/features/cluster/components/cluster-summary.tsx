import { BoxIcon, CpuIcon, MemoryStickIcon, ServerIcon } from "lucide-react"

import { MetricTile } from "@/components/common/metric-tile"
import { UsageBar } from "@/components/common/usage-bar"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatBytes, formatCores, formatPercent } from "@/lib/format"
import type { ClusterTotals } from "@/lib/types"

/** Totals of all nodes: counts, capacity, usage and the CPU models in the cluster. */
export function ClusterSummary({ totals: t }: { totals: ClusterTotals }) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <MetricTile
          icon={<ServerIcon />}
          label="Nodes"
          value={`${t.readyNodes} / ${t.nodes}`}
          hint="ready"
          accent={t.readyNodes === t.nodes}
        />
        <MetricTile
          icon={<CpuIcon />}
          label="CPU"
          value={formatCores(t.cpuCapacityMillis)}
          hint={t.physicalCores ? `${t.physicalCores} physical` : undefined}
        />
        <MetricTile icon={<MemoryStickIcon />} label="Memory" value={formatBytes(t.memoryCapacityBytes, 0)} />
        <MetricTile icon={<BoxIcon />} label="Pods" value={`${t.podsRunning} / ${t.podsCapacity}`} hint="running" />
      </div>
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle>Capacity</CardTitle>
            <CardDescription>
              {t.metricsAvailable
                ? "Live usage from metrics-server and resources requested by running pods."
                : "metrics-server is not available, only requests are shown."}
            </CardDescription>
          </div>
          <CapacityLegend />
        </CardHeader>
        <CardContent className="grid gap-6 md:grid-cols-2">
          <Row
            label="CPU"
            used={t.metricsAvailable ? t.cpuUsedMillis : null}
            requested={t.cpuRequestedMillis}
            capacity={t.cpuCapacityMillis}
            fmt={formatCores}
          />
          <Row
            label="Memory"
            used={t.metricsAvailable ? t.memoryUsedBytes : null}
            requested={t.memoryRequestedBytes}
            capacity={t.memoryCapacityBytes}
            fmt={(v) => formatBytes(v)}
          />
          {t.cpuModels.length > 0 && (
            <div className="flex flex-wrap items-center gap-2 md:col-span-2">
              <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Processors</span>
              {t.cpuModels.map((m) => (
                <Badge key={m.model} variant="secondary" className="gap-1.5 py-1">
                  <span className="font-mono text-emerald-600 dark:text-emerald-400">{m.count}×</span>
                  {m.model}
                </Badge>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function Row({
  label,
  used,
  requested,
  capacity,
  fmt,
}: {
  label: string
  used: number | null
  requested: number
  capacity: number
  fmt: (v: number) => string
}) {
  return (
    <div className="space-y-2">
      <div className="flex flex-col gap-0.5 text-sm sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
        <span className="font-medium">{label}</span>
        <span className="font-mono text-xs text-muted-foreground tabular-nums sm:text-right">
          {used != null && (
            <span className="text-foreground">
              {fmt(used)} used ({formatPercent(used, capacity)}) ·{" "}
            </span>
          )}
          {fmt(requested)} requested ({formatPercent(requested, capacity)}) of {fmt(capacity)}
        </span>
      </div>
      <UsageBar value={used} requested={requested} max={capacity} />
    </div>
  )
}

/** Legend for the capacity bars (used, requested). */
function CapacityLegend() {
  return (
    <div className="flex items-center gap-4 text-xs text-muted-foreground">
      <span className="flex items-center gap-1.5">
        <span className="size-2 rounded-full bg-emerald-500" />
        used
      </span>
      <span className="flex items-center gap-1.5">
        <span className="size-2 rounded-full bg-sky-500/40" />
        requested by pods
      </span>
    </div>
  )
}
