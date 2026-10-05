import { LockIcon } from "lucide-react"

import type { Phase } from "@/lib/types"
import { cn } from "@/lib/utils"

const styles: Record<Phase, { label: string; dot: string; ring: string; pulse?: boolean }> = {
  Running: {
    label: "Running",
    dot: "bg-emerald-500",
    ring: "text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 ring-emerald-500/20",
  },
  Starting: {
    label: "Starting",
    dot: "bg-amber-500",
    ring: "text-amber-600 dark:text-amber-400 bg-amber-500/10 ring-amber-500/20",
    pulse: true,
  },
  Stopping: {
    label: "Stopping",
    dot: "bg-orange-500",
    ring: "text-orange-600 dark:text-orange-400 bg-orange-500/10 ring-orange-500/20",
    pulse: true,
  },
  Installing: {
    label: "Installing",
    dot: "bg-sky-500",
    ring: "text-sky-600 dark:text-sky-400 bg-sky-500/10 ring-sky-500/20",
    pulse: true,
  },
  InstallFailed: {
    label: "Install failed",
    dot: "bg-red-500",
    ring: "text-red-600 dark:text-red-400 bg-red-500/10 ring-red-500/20",
  },
  Offline: {
    label: "Offline",
    dot: "bg-zinc-400 dark:bg-zinc-500",
    ring: "text-muted-foreground bg-muted ring-border",
  },
  Pending: { label: "Pending", dot: "bg-zinc-400", ring: "text-muted-foreground bg-muted ring-border", pulse: true },
}

/** Colored dot for a server phase (sidebar). */
export function StatusDot({ phase, className }: { phase: Phase; className?: string }) {
  const s = styles[phase] ?? styles.Pending
  return (
    <span className={cn("relative inline-flex size-2 shrink-0", className)}>
      {(s.pulse || phase === "Running") && (
        <span
          className={cn(
            "absolute inline-flex size-full rounded-full opacity-60",
            s.dot,
            s.pulse ? "animate-ping" : "animate-[ping_2.5s_ease-out_infinite]",
          )}
        />
      )}
      <span className={cn("relative inline-flex size-2 rounded-full", s.dot)} />
    </span>
  )
}

/** Badge with the phase of a server (Running, Offline, …). */
export function StatusBadge({ phase, className }: { phase: Phase; className?: string }) {
  const s = styles[phase] ?? styles.Pending
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset",
        s.ring,
        className,
      )}
    >
      <StatusDot phase={phase} />
      {s.label}
    </span>
  )
}

/** Marks a server an administrator has suspended. */
export function SuspendedBadge({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full bg-red-500/10 px-2.5 py-0.5 text-xs font-medium text-red-600 ring-1 ring-red-500/20 ring-inset dark:text-red-400",
        className,
      )}
    >
      <LockIcon className="size-3" />
      Suspended
    </span>
  )
}
