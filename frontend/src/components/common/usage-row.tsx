import { UsageBar } from "@/components/common/usage-bar"

/** A labelled usage bar of a card (server, node): icon and label left, the numbers right. */
export function UsageRow({
  icon,
  label,
  detail,
  value,
  requested,
  max,
}: {
  icon: React.ReactNode
  label: string
  detail: React.ReactNode
  value: number | null | undefined
  requested?: number
  max: number | null | undefined
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-muted-foreground [&_svg]:size-3.5">
        <span className="flex items-center gap-1.5">
          {icon}
          {label}
        </span>
        <span className="font-mono tabular-nums">{detail}</span>
      </div>
      <UsageBar value={value} requested={requested} max={max} />
    </div>
  )
}
