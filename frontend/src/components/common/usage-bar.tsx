import { Progress } from "@/components/ui/progress"
import { cn } from "@/lib/utils"

// Styles of the shadcn track and indicator, written out in full so Tailwind finds them; the track fills the bar.
const bar = "[&_[data-slot=progress-track]]:h-full [&_[data-slot=progress-indicator]]:duration-700"

/**
 * The shadcn progress bar as a usage bar that turns amber/red when the value gets close to the limit. With
 * `requested`, a lighter bar behind it shows the resources requested by running pods (what the scheduler has
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
  const color =
    used > 90
      ? "[&_[data-slot=progress-indicator]]:bg-destructive"
      : used > 75
        ? "[&_[data-slot=progress-indicator]]:bg-warning"
        : "[&_[data-slot=progress-indicator]]:bg-success"
  return (
    <div className={cn("relative w-full", requested == null ? "h-1.5" : "h-2", className)}>
      {requested != null && (
        <Progress
          aria-hidden
          value={percent(requested)}
          className={cn("absolute inset-0", bar, "[&_[data-slot=progress-indicator]]:bg-info/30")}
        />
      )}
      <Progress
        value={used}
        className={cn(
          "absolute inset-0",
          bar,
          color,
          requested != null && "[&_[data-slot=progress-track]]:bg-transparent",
        )}
      />
    </div>
  )
}
