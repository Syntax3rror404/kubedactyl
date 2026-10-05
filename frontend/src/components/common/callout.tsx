import { cn } from "@/lib/utils"

const tones = {
  info: { box: "border-sky-500/30 bg-sky-500/5", icon: "text-sky-500" },
  warning: { box: "border-amber-500/30 bg-amber-500/5", icon: "text-amber-500" },
  danger: { box: "border-red-500/30 bg-red-500/5", icon: "text-red-500" },
  success: { box: "border-emerald-500/30 bg-emerald-500/5", icon: "text-emerald-500" },
}

/** A colored notice: icon, optional bold title, text and an action on the right (below on phones). */
export function Callout({
  tone,
  icon,
  title,
  children,
  action,
  progress,
  className,
}: {
  tone: keyof typeof tones
  icon?: React.ReactNode
  title?: React.ReactNode
  children?: React.ReactNode
  action?: React.ReactNode
  // Shows the animated stripe at the top while something is in progress.
  progress?: boolean
  className?: string
}) {
  return (
    <div
      className={cn(
        "relative flex flex-col gap-3 overflow-hidden rounded-xl border px-4 py-3 text-sm sm:flex-row sm:items-center",
        tones[tone].box,
        className,
      )}
    >
      {progress && <ProgressStripe />}
      <div className="flex min-w-0 flex-1 items-start gap-2">
        {icon && <span className={cn("mt-0.5 shrink-0 [&_svg]:size-4", tones[tone].icon)}>{icon}</span>}
        <div className="min-w-0 flex-1 space-y-1">
          {title && <p className="font-medium">{title}</p>}
          {children}
        </div>
      </div>
      {action}
    </div>
  )
}

/** Animated stripe along the top edge of a box (relative, overflow-hidden) while something runs. */
export function ProgressStripe() {
  return (
    <div className="absolute inset-x-0 top-0 h-0.5 animate-pulse bg-gradient-to-r from-transparent via-sky-500 to-transparent" />
  )
}
