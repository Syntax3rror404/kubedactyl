import { Area, AreaChart, YAxis } from "recharts"

import { ChartContainer, type ChartConfig } from "@/components/ui/chart"
import type { Sample } from "@/features/servers/hooks/use-samples"

/** Small area chart of the recent samples of a stat tile. */
export function StatChart({
  label,
  data,
  dataKey,
  color,
  max,
}: {
  label: string
  data: Sample[]
  dataKey: "cpu" | "mem"
  color: string
  max?: number
}) {
  const config = { [dataKey]: { label: label, color: color } } satisfies ChartConfig
  const id = `fill-${dataKey}`
  return (
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
  )
}
