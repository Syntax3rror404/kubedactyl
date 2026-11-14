import { LockIcon, Trash2Icon } from "lucide-react"

import { NavDot } from "@/base/navigation/nav-badge"
import type { NavTone } from "@/base/types"
import { Badge } from "@/components/ui/badge"
import { phaseStatus } from "@/lib/format"
import type { Phase } from "@/lib/types"
import { cn } from "@/lib/utils"

const badgeTone: Record<NavTone, string> = {
  success: "text-success bg-success/10 border-success/20",
  warning: "text-warning bg-warning/10 border-warning/20",
  pending: "text-pending bg-pending/10 border-pending/20",
  danger: "text-destructive bg-destructive/10 border-destructive/20",
  info: "text-info bg-info/10 border-info/20",
  muted: "text-muted-foreground bg-muted border-border",
}

/** Badge with the phase of a server (Running, Offline, …). */
export function StatusBadge({ phase, className }: { phase: Phase; className?: string }) {
  const s = phaseStatus[phase] ?? phaseStatus.Pending
  return (
    <Badge variant="outline" className={cn("gap-1.5", badgeTone[s.tone], className)}>
      <NavDot tone={s.tone} pulse={s.pulse} />
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
