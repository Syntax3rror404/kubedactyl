import { Progress } from "@/components/ui/progress"
import { cn } from "@/lib/utils"

// A light sweeps over the bar (it is 40 % wide, so -100 % and 200 % are outside); without a value a third of the
// bar slides through (animations win over the inline transform of the shadcn indicator).
const keyframes = `@keyframes progress-shine { from { background-position: -100% 0 } to { background-position: 200% 0 } }
@keyframes progress-slide { from { transform: translateX(-100%) } to { transform: translateX(300%) } }`

// Styles of the shadcn indicator, written out in full so Tailwind finds them.
const indicator = [
  "[&>[data-slot=progress-indicator]]:bg-sky-500",
  "[&>[data-slot=progress-indicator]]:bg-[linear-gradient(90deg,transparent,rgb(255_255_255/0.45),transparent)]",
  "[&>[data-slot=progress-indicator]]:bg-[length:40%_100%]",
  "[&>[data-slot=progress-indicator]]:bg-no-repeat",
  "[&>[data-slot=progress-indicator]]:duration-700",
  "[&>[data-slot=progress-indicator]]:ease-out",
  "[&>[data-slot=progress-indicator]]:animate-[progress-shine_1.6s_ease-in-out_infinite]",
  "motion-reduce:[&>[data-slot=progress-indicator]]:animate-none",
].join(" ")

const sliding = [
  "[&>[data-slot=progress-indicator]]:max-w-1/3",
  "[&>[data-slot=progress-indicator]]:rounded-full",
  "[&>[data-slot=progress-indicator]]:animate-[progress-slide_1.4s_ease-in-out_infinite]",
  "motion-reduce:[&>[data-slot=progress-indicator]]:translate-x-0",
].join(" ")

/**
 * The shadcn progress bar in sky with a light sweeping over it, for work that runs; without a value a part of it
 * slides through, for work whose end is not known (still for users who prefer reduced motion).
 *
 * Usage:
 *   <ProgressBar value={42} />
 *   <ProgressBar value={done} max={total} className="h-1.5" />
 *   <ProgressBar />
 */
export function ProgressBar({ value, max = 100, className }: { value?: number; max?: number; className?: string }) {
  const percent = value == null ? undefined : max > 0 ? Math.min(100, Math.max(0, (value / max) * 100)) : 0
  return (
    <>
      <style href="progress-bar" precedence="default">
        {keyframes}
      </style>
      <Progress value={percent ?? null} className={cn("h-2", indicator, percent == null && sliding, className)} />
    </>
  )
}
