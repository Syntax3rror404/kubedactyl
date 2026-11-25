import { ArrowUpIcon } from "lucide-react"

import { BRAND_FROM, BRAND_TO } from "@/app/config"
import type { NavBadge, NavTone } from "@/base/types"
import { ShineBorder } from "@/components/ui/shine-border"
import { cn } from "@/lib/utils"

const dot: Record<NavTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  pending: "bg-pending",
  danger: "bg-destructive",
  info: "bg-info",
  muted: "bg-muted-foreground",
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

/** A status dot set apart from the icon it sits on. */
function NavMark({ tone, className }: { tone: NavTone; className?: string }) {
  return <NavDot tone={tone} className={cn("ring-2 ring-sidebar", className)} />
}

/**
 * A mark on the bottom right corner of an icon. Collapsed to icons the button cuts off what sticks out, so the mark
 * moves inside.
 */
export function NavCornerMark({ tone }: { tone: NavTone }) {
  return (
    <NavMark
      tone={tone}
      className={cn(
        "absolute -right-0.5 -bottom-0.5",
        "group-data-[collapsible=icon]:right-0.5 group-data-[collapsible=icon]:bottom-0.5",
      )}
    />
  )
}

/** The light circling an update's edge in the app's brand colors (6 s a round). */
function UpdateShine() {
  return <ShineBorder shineColor={[BRAND_FROM, BRAND_TO]} duration={6} />
}

/** What stays of a badge on the icon when the sidebar is collapsed to icons; plain text leaves nothing. */
export function NavBadgeMark({ badge }: { badge: NavBadge }) {
  if (badge.kind === "text") return null
  if (badge.kind !== "update") return <NavMark tone={badge.tone} />
  return (
    <span className="relative flex size-3.5 items-center justify-center rounded-full bg-sidebar text-success">
      <ArrowUpIcon className="size-2.5!" strokeWidth={3} />
      <UpdateShine />
    </span>
  )
}

/** The sign at the end of a sidebar entry: a count, an update, a status dot with optional text, or plain text. */
export function NavBadgeView({ badge, className }: { badge: NavBadge; className?: string }) {
  if (badge.kind === "count") {
    return (
      <span className={cn("rounded-full px-1.5 text-xs font-medium tabular-nums", pill[badge.tone], className)}>
        {badge.value}
      </span>
    )
  }
  if (badge.kind === "update") {
    return (
      <span
        className={cn(
          "relative flex h-5 items-center gap-1 rounded-full px-1.5 text-xs font-medium tabular-nums",
          pill.success,
          className,
        )}
      >
        <ArrowUpIcon className="size-3! shrink-0" strokeWidth={2.5} />
        {badge.version}
        <UpdateShine />
      </span>
    )
  }
  if (badge.kind === "text") {
    return <span className={cn("text-xs text-muted-foreground tabular-nums", className)}>{badge.text}</span>
  }
  return (
    <span className={cn("flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground tabular-nums", className)}>
      {badge.text}
      <NavDot tone={badge.tone} pulse={badge.pulse} />
    </span>
  )
}
