import { LockIcon, Trash2Icon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import type { Phase } from "@/lib/types"
import { cn } from "@/lib/utils"

const styles: Record<Phase, { label: string; dot: string; badge: string; pulse?: boolean }> = {
  Running: {
    label: "Running",
    dot: "bg-success",
    badge: "text-success bg-success/10 border-success/20",
  },
  Starting: {
    label: "Starting",
    dot: "bg-warning",
    badge: "text-warning bg-warning/10 border-warning/20",
    pulse: true,
  },
  Stopping: {
    label: "Stopping",
    dot: "bg-orange-500",
    badge: "text-orange-600 dark:text-orange-400 bg-orange-500/10 border-orange-500/20",
    pulse: true,
  },
  Installing: {
    label: "Installing",
    dot: "bg-info",
    badge: "text-info bg-info/10 border-info/20",
    pulse: true,
  },
  InstallFailed: {
    label: "Install failed",
    dot: "bg-destructive",
    badge: "text-destructive bg-destructive/10 border-destructive/20",
  },
  Offline: {
    label: "Offline",
    dot: "bg-zinc-400 dark:bg-zinc-500",
    badge: "text-muted-foreground bg-muted border-border",
  },
  Pending: { label: "Pending", dot: "bg-zinc-400", badge: "text-muted-foreground bg-muted border-border", pulse: true },
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
    <Badge variant="outline" className={cn("gap-1.5", s.badge, className)}>
      <StatusDot phase={phase} />
      {s.label}
    </Badge>
  )
}

/** Marks a server an administrator has suspended. */
export function SuspendedBadge({ className }: { className?: string }) {
  return (
    <LockBadge icon={<LockIcon />} className={className}>
      Suspended
    </LockBadge>
  )
}

/** Marks a server that is being removed (it stays until its volume is released). */
export function RemovingBadge({ className }: { className?: string }) {
  return (
    <LockBadge icon={<Trash2Icon />} className={className}>
      Removal scheduled
    </LockBadge>
  )
}

function LockBadge({ icon, className, children }: { icon: React.ReactNode; className?: string; children: string }) {
  return (
    <Badge variant="destructive" className={className}>
      {icon}
      {children}
    </Badge>
  )
}
