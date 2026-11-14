import type { ComponentType } from "react"

import type { NavBadge, NavTone } from "@/base/types"
import { cn } from "@/lib/utils"

const dot: Record<NavTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  pending: "bg-pending",
  danger: "bg-destructive",
  info: "bg-info",
  muted: "bg-muted-foreground",
}

const color: Record<NavTone, string> = {
  success: "text-success",
  warning: "text-warning",
  pending: "text-pending",
  danger: "text-destructive",
  info: "text-info",
  muted: "text-muted-foreground",
}

const pill: Record<NavTone, string> = {
  success: "bg-success/15 text-success",
  warning: "bg-warning/15 text-warning",
  pending: "bg-pending/15 text-pending",
  danger: "bg-destructive/15 text-destructive",
  info: "bg-info/15 text-info",
  muted: "bg-muted text-muted-foreground",
}

/** A status dot; pulse for something in progress. */
export function NavDot({ tone, pulse, className }: { tone: NavTone; pulse?: boolean; className?: string }) {
  return (
    <span className={cn("relative inline-flex size-2 shrink-0 rounded-full", className)}>
      {pulse && (
        <span className={cn("absolute inline-flex size-full animate-ping rounded-full opacity-60", dot[tone])} />
      )}
      <span className={cn("relative inline-flex size-2 rounded-full", dot[tone])} />
    </span>
  )
}

type Icon = ComponentType<{ className?: string }>

/**
 * A status dot, or the icon given instead, set apart from the icon it sits on. The icon needs size-3!: the sidebar
 * buttons size every svg inside them.
 */
export function NavMark({ tone, icon: Icon, className }: { tone: NavTone; icon?: Icon; className?: string }) {
  if (Icon) return <Icon className={cn("size-3! rounded-full bg-sidebar", color[tone], className)} />
  return <NavDot tone={tone} className={cn("ring-2 ring-sidebar", className)} />
}

/**
 * A mark on the bottom right corner of an icon. Collapsed to icons the button cuts off what sticks out, so the mark
 * moves inside.
 */
export function NavCornerMark({ tone, icon }: { tone: NavTone; icon?: Icon }) {
  return (
    <NavMark
      tone={tone}
      icon={icon}
      className={cn(
        "absolute -right-0.5 -bottom-0.5",
        "group-data-[collapsible=icon]:right-0.5 group-data-[collapsible=icon]:bottom-0.5",
      )}
    />
  )
}

/** The sign at the end of a sidebar entry: a count, a status dot (or icon) with optional text, or plain text. */
export function NavBadgeView({ badge, className }: { badge: NavBadge; className?: string }) {
  if (badge.kind === "count") {
    return (
      <span className={cn("rounded-full px-1.5 text-xs font-medium tabular-nums", pill[badge.tone], className)}>
        {badge.value}
      </span>
    )
  }
  if (badge.kind === "text") {
    return <span className={cn("text-xs text-muted-foreground tabular-nums", className)}>{badge.text}</span>
  }
  return (
    <span className={cn("flex shrink-0 items-center gap-1.5 font-mono text-xs text-muted-foreground", className)}>
      {badge.text}
      {badge.icon ? (
        <badge.icon className={cn("size-3! shrink-0", color[badge.tone])} />
      ) : (
        <NavDot tone={badge.tone} pulse={badge.pulse} />
      )}
    </span>
  )
}
