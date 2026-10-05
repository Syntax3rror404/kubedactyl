import { Area, AreaChart, YAxis } from "recharts"

import { ChartContainer, type ChartConfig } from "@/components/ui/chart"
import type { Sample } from "@/features/servers/hooks/use-samples"

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
  const config = { [dataKey]: { label: label, color: color } } satisfies ChartConfig
  const id = `fill-${dataKey}`
  return (
    <StatTile icon={icon} label={label} value={value} limit={limit}>
      {!active || data.length < 2 ? (
        <div className="mt-3 flex h-16 items-end">
          <div className="w-full border-t border-dashed" />
        </div>
      ) : (
        <ChartContainer config={config} className="mt-3 aspect-auto h-16 w-full">
          <AreaChart data={data} margin={{ top: 2, right: 0, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.45} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <YAxis hide domain={[0, max ?? "auto"]} />
            <Area
              dataKey={dataKey}
              type="monotone"
              stroke={color}
              strokeWidth={2}
              fill={`url(#${id})`}
              isAnimationActive={false}
            />
          </AreaChart>
        </ChartContainer>
      )}
    </StatTile>
  )
}
