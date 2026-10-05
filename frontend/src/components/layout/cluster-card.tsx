import { useState, type ReactNode } from "react"
import { ChevronRightIcon } from "lucide-react"
import { Link } from "react-router"

import { CheckStatusIcon } from "@/components/common/check-status-icon"
import { useClusterInfo, useClusterHealth } from "@/lib/queries"
import type { CheckStatus } from "@/lib/types"
import { cn } from "@/lib/utils"

const dot: Record<CheckStatus, string> = {
  ok: "bg-emerald-500",
  warning: "bg-amber-500",
  error: "bg-red-500",
  skipped: "bg-muted-foreground",
}

/** Devices without hover (phones, tablets): the card opens and closes on tap instead. */
const touch = () => typeof window !== "undefined" && window.matchMedia("(hover: none)").matches

/**
 * Sidebar footer: the cluster the panel is connected to. Problems are always listed;
 * hovering the card (tapping it on touch devices) also shows the passing checks (green).
 */
export function ClusterCard() {
  const cluster = useClusterInfo()
  const health = useClusterHealth()
  const [tapToOpen] = useState(touch)
  const [open, setOpen] = useState(false)
  if (!cluster.data) return null
  const c = cluster.data
  const status = health.data?.status ?? "ok"
  const checks = health.data?.checks ?? []
  const hasProblems = checks.some((check) => check.status === "warning" || check.status === "error")
  const className = cn(
    "group/cluster block w-full rounded-lg border bg-sidebar-accent/40 p-3 text-left text-xs transition-colors group-data-[collapsible=icon]:hidden hover:bg-sidebar-accent",
    status === "warning" && "border-amber-500/40",
    status === "error" && "border-red-500/50",
  )
  const frame = (children: ReactNode) =>
    tapToOpen ? (
      <div
        role="button"
        tabIndex={0}
        aria-expanded={open}
        data-open={open}
        className={className}
        onClick={() => setOpen(!open)}
        onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && setOpen(!open)}
      >
        {children}
        {open && (
          <Link
            to="/cluster"
            onClick={(e) => e.stopPropagation()}
            className="mt-2 flex items-center justify-end gap-1 font-medium text-foreground"
          >
            Cluster details
            <ChevronRightIcon className="size-3.5" />
          </Link>
        )}
      </div>
    ) : (
      <Link to="/cluster" title="Cluster details" className={className}>
        {children}
      </Link>
    )
  return frame(
    <>
      <div className="mb-2 flex items-center gap-2 font-medium">
        <span className="relative flex size-2">
          <span className={cn("absolute inline-flex size-full animate-ping rounded-full opacity-50", dot[status])} />
          <span className={cn("relative inline-flex size-2 rounded-full", dot[status])} />
        </span>
        <span className="truncate">{c.context}</span>
      </div>
      {checks.length > 0 && (
        <ul
          className={cn(
            "mb-2 space-y-1.5",
            !hasProblems && "hidden group-hover/cluster:block group-data-[open=true]/cluster:block",
          )}
        >
          {checks.map((check) => (
            <li
              key={check.id}
              className={cn(
                "flex gap-1.5",
                (check.status === "ok" || check.status === "skipped") &&
                  "hidden group-hover/cluster:flex group-data-[open=true]/cluster:flex",
              )}
              title={check.message}
            >
              <CheckStatusIcon status={check.status} className="mt-px size-3.5" />
              <span className="min-w-0">
                <span className="font-medium text-foreground">{check.label}</span>
                {check.message && <span className="line-clamp-2 text-muted-foreground">{check.message}</span>}
              </span>
            </li>
          ))}
        </ul>
      )}
      <dl className="grid grid-cols-[auto_1fr] gap-x-2 gap-y-1 text-muted-foreground">
        <dt>Version</dt>
        <dd className="truncate text-right font-mono">{c.version}</dd>
        <dt>Namespace</dt>
        <dd className="truncate text-right font-mono">{c.namespace}</dd>
        <dt>Storage</dt>
        <dd className="truncate text-right font-mono">{c.storageClass}</dd>
        <dt>LB pool</dt>
        <dd className="truncate text-right font-mono">{c.loadBalancerPool}</dd>
      </dl>
    </>,
  )
}
