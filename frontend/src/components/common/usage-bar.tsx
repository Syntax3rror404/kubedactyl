import { cn } from "@/lib/utils"

/**
 * A thin usage bar that turns amber/red when the value gets close to the limit. With `requested`, a
 * lighter part behind it shows the resources requested by running pods (what the scheduler has
 * already handed out).
 */
export function UsageBar({
  value,
  requested,
  max,
  className,
}: {
  value: number | null | undefined
  requested?: number
  max: number | null | undefined
  className?: string
}) {
  const percent = (v: number | null | undefined) => (v != null && max ? Math.min(100, (v / max) * 100) : 0)
  const used = percent(value)
  const color = used > 90 ? "bg-red-500" : used > 75 ? "bg-amber-500" : "bg-emerald-500"
  return (
    <div
      className={cn(
        "relative w-full overflow-hidden rounded-full bg-muted",
        requested == null ? "h-1.5" : "h-2",
        className,
      )}
    >
      {requested != null && (
        <div
          className="absolute inset-y-0 left-0 rounded-full bg-sky-500/30 transition-all duration-700"
          style={{ width: `${percent(requested)}%` }}
        />
      )}
      <div
        className={cn("absolute inset-y-0 left-0 rounded-full transition-all duration-700 ease-out", color)}
        style={{ width: `${used}%` }}
      />
    </div>
  )
}
