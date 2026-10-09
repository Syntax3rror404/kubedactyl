import { AlertTriangleIcon, NetworkIcon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { UsageBar } from "@/components/common/usage-bar"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { DefaultCell, type Selection, type SelectionHandlers } from "@/features/settings/components/default-cell"
import { formatCount, formatPoolSize, poolUsage } from "@/lib/format"
import type { Pool } from "@/lib/types"

/**
 * All Cilium LB IPAM pools; the checked ones can be selected for servers. "Allow ClusterIP" lets
 * owners publish a server only inside the cluster.
 */
export function PoolsCard({
  pools,
  ipv6Missing,
  loadError,
  fieldError,
  selection,
  allowClusterIP,
  onAllowClusterIP,
  ...handlers
}: {
  pools?: Pool[]
  /** Why servers get no IPv6 address even from a pool with an IPv6 block. */
  ipv6Missing?: string
  loadError?: string
  fieldError?: string
  selection: Selection
  allowClusterIP: boolean
  onAllowClusterIP: (allow: boolean) => void
} & SelectionHandlers) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <NetworkIcon className="size-4" />
          Load balancer pools
        </CardTitle>
        <CardDescription>
          Checked Cilium pools can be picked for servers. Users can move their servers between them.
        </CardDescription>
      </CardHeader>
      <CardContent className="px-0">
        {loadError ? (
          <Callout tone="warning" icon={<AlertTriangleIcon />} className="mx-6">
            <p>{loadError}</p>
          </Callout>
        ) : !pools ? (
          <Skeleton className="mx-6 h-32" />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10 pl-6" />
                <TableHead>Pool</TableHead>
                <TableHead className="hidden md:table-cell">Addresses</TableHead>
                <TableHead className="w-40">Free</TableHead>
                <TableHead className="hidden lg:table-cell">Service labels</TableHead>
                <TableHead className="w-32 pr-6 text-right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {pools.map((p) => (
                <PoolRow key={p.name} pool={p} ipv6Missing={ipv6Missing} selection={selection} {...handlers} />
              ))}
            </TableBody>
          </Table>
        )}
        {fieldError && <FieldError className="mx-6 mt-3">{fieldError}</FieldError>}
        <Field orientation="horizontal" className="mx-6 mt-6 w-auto">
          <Switch id="cluster-ip" checked={allowClusterIP} onCheckedChange={onAllowClusterIP} />
          <FieldLabel htmlFor="cluster-ip">Allow ClusterIP</FieldLabel>
        </Field>
      </CardContent>
    </Card>
  )
}

function PoolRow({
  pool: p,
  ipv6Missing,
  selection,
  onToggle,
  onDefault,
}: { pool: Pool; ipv6Missing?: string; selection: Selection } & SelectionHandlers) {
  const on = selection.enabled.includes(p.name)
  const labels = Object.entries(p.serviceLabels ?? {})
  const usage = poolUsage(p)
  const checkbox = (
    <Checkbox
      checked={on}
      disabled={!p.selectable && !on}
      onCheckedChange={(v) => onToggle(p.name, v === true)}
      aria-label={`Enable ${p.name}`}
    />
  )
  return (
    <TableRow data-state={on ? "selected" : undefined} className={!p.selectable ? "text-muted-foreground" : undefined}>
      <TableCell className="pl-6">
        {p.selectable ? (
          checkbox
        ) : (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>{checkbox}</span>
            </TooltipTrigger>
            <TooltipContent>{p.reason}</TooltipContent>
          </Tooltip>
        )}
      </TableCell>
      <TableCell>
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-mono text-sm font-medium">{p.name}</span>
          {p.disabled && <Badge variant="outline">disabled</Badge>}
          {p.conflict && <Badge variant="destructive">conflict</Badge>}
        </div>
        {ipv6Missing && usage.some((u) => u.family === "IPv6") && (
          <div className="mt-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="outline" className="border-amber-500/40 text-amber-500">
                  IPv6 not available in cluster
                </Badge>
              </TooltipTrigger>
              <TooltipContent>{ipv6Missing}</TooltipContent>
            </Tooltip>
          </div>
        )}
        {p.reason && <p className="mt-0.5 text-xs text-muted-foreground">{p.reason}</p>}
      </TableCell>
      <TableCell className="hidden font-mono text-xs md:table-cell">
        {(p.blocks ?? []).map((b) => (
          <div key={b}>{b}</div>
        ))}
      </TableCell>
      <TableCell>
        {usage.length ? (
          <div className="space-y-2">
            {usage.map((u) => (
              <div key={u.family ?? "all"} className="space-y-1">
                <div className="font-mono text-xs tabular-nums">
                  {u.family && <span className="text-muted-foreground">{u.family} </span>}
                  {u.family === "IPv6" ? (
                    <>
                      {formatCount(u.used)}{" "}
                      <span className="text-muted-foreground">used of {formatPoolSize(u.total)}</span>
                    </>
                  ) : (
                    <>
                      {formatCount(u.available)} <span className="text-muted-foreground">/ {formatCount(u.total)}</span>
                    </>
                  )}
                </div>
                {/* An IPv6 block never runs out: its bar would always be empty. */}
                {u.family !== "IPv6" && <UsageBar value={u.used} max={u.total} />}
              </div>
            ))}
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">unknown</span>
        )}
      </TableCell>
      <TableCell className="hidden lg:table-cell">
        {labels.length ? (
          labels.map(([k, v]) => (
            <code key={k} className="block font-mono text-xs">
              {k}={v}
            </code>
          ))
        ) : (
          <span className="text-xs">-</span>
        )}
      </TableCell>
      <TableCell className="pr-6 text-right">
        <DefaultCell name={p.name} selection={selection} onDefault={onDefault} />
      </TableCell>
    </TableRow>
  )
}
