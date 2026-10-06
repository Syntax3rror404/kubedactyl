import { lazy, Suspense } from "react"

import type { Sample } from "@/features/servers/hooks/use-samples"

// The chart library is large: loaded on its own, the console does not wait for it.
const StatChart = lazy(() => import("@/features/servers/components/stat-chart").then((m) => ({ default: m.StatChart })))

// Shown while there is nothing to draw (server offline, fewer than two samples) and while the chart loads.
const noChart = (
  <div className="mt-3 flex h-16 items-end">
    <div className="w-full border-t border-dashed" />
  </div>
)

/** One value of the stats panel (CPU, memory, disk, uptime). */
export function StatTile({
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
export function ChartStatTile({
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
