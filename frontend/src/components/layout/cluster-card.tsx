import { Fragment } from "react"
import { ChevronRightIcon } from "lucide-react"
import { Link } from "react-router"

import { CheckStatusIcon } from "@/components/common/check-status-icon"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { useClusterInfo, useClusterHealth } from "@/lib/queries"
import type { Check, CheckStatus } from "@/lib/types"
import { cn } from "@/lib/utils"

const dot: Record<CheckStatus, string> = {
  ok: "bg-success",
  warning: "bg-warning",
  error: "bg-destructive",
  skipped: "bg-muted-foreground",
}

const isProblem = (check: Check) => check.status === "warning" || check.status === "error"

/**
 * Sidebar footer: the cluster the panel is connected to. Problems are always listed; a click opens the card
 * upwards with the passing checks and the link to the cluster page.
 */
export function ClusterCard() {
  const cluster = useClusterInfo()
  const health = useClusterHealth()
  if (!cluster.data) return null
  const c = cluster.data
  const status = health.data?.status ?? "ok"
  const checks = health.data?.checks ?? []
  const problems = checks.filter(isProblem)
  const passing = checks.filter((check) => !isProblem(check))
  return (
    <Collapsible
      className={cn(
        "rounded-lg border bg-sidebar-accent/40 text-xs transition-colors group-data-[collapsible=icon]:hidden hover:bg-sidebar-accent",
        status === "warning" && "border-warning/40",
        status === "error" && "border-destructive/50",
      )}
    >
      <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-out data-ending-style:h-0 data-starting-style:h-0">
        <div className="space-y-2 border-b p-3">
          {passing.length > 0 && <CheckList checks={passing} />}
          <Link to="/cluster" className="flex items-center justify-end gap-1 font-medium hover:underline">
            Cluster details
            <ChevronRightIcon className="size-3.5" />
          </Link>
        </div>
      </CollapsibleContent>
      <CollapsibleTrigger nativeButton={false} render={<div />} className="block w-full cursor-pointer p-3 text-left">
        <div className="mb-2 flex items-center gap-2 font-medium">
          <span className="relative flex size-2">
            <span className={cn("absolute inline-flex size-full animate-ping rounded-full opacity-50", dot[status])} />
            <span className={cn("relative inline-flex size-2 rounded-full", dot[status])} />
          </span>
          <span className="truncate">{c.context}</span>
        </div>
        {problems.length > 0 && <CheckList checks={problems} className="mb-2" />}
        <dl className="grid grid-cols-[auto_1fr] gap-x-2 gap-y-1 text-muted-foreground">
          {(
            [
              ["Version", c.version],
              ["Namespace", c.namespace],
              ["Storage", c.storageClass],
              ["LB pool", c.loadBalancerPool],
            ] as const
          ).map(([label, value]) => (
            <Fragment key={label}>
              <dt>{label}</dt>
              <dd className="truncate text-right font-mono">{value}</dd>
            </Fragment>
          ))}
        </dl>
      </CollapsibleTrigger>
    </Collapsible>
  )
}

function CheckList({ checks, className }: { checks: Check[]; className?: string }) {
  return (
    <ul className={cn("space-y-1.5", className)}>
      {checks.map((check) => (
        <li key={check.id} className="flex gap-1.5" title={check.message}>
          <CheckStatusIcon status={check.status} className="mt-px size-3.5" />
          <span className="min-w-0">
            <span className="font-medium text-foreground">{check.label}</span>
            {check.message && <span className="line-clamp-2 text-muted-foreground">{check.message}</span>}
          </span>
        </li>
      ))}
    </ul>
  )
}
