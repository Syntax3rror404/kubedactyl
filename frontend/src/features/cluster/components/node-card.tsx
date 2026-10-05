import { AlertTriangleIcon, CpuIcon, HardDriveIcon, LoaderCircleIcon, MemoryStickIcon, MonitorIcon } from "lucide-react"

import { GlowCard } from "@/components/common/glow-card"
import { UsageRow } from "@/components/common/usage-row"
import { Badge } from "@/components/ui/badge"
import { formatBytes, formatCores, formatPercent, formatRelativeTime } from "@/lib/format"
import type { ClusterNode } from "@/lib/types"
import { cn } from "@/lib/utils"

/** One node: status, hardware, capacity bars and system details (glowing border on hover). */
export function NodeCard({ node: n }: { node: ClusterNode }) {
  return (
    <GlowCard>
      <div className={cn("flex h-full flex-col gap-4 p-5", !n.ready && "opacity-80")}>
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate font-mono font-semibold">{n.name}</h3>
            <p className="font-mono text-xs text-muted-foreground">{n.internalIP}</p>
          </div>
          <div className="flex flex-wrap justify-end gap-1.5">
            {n.roles.map((r) => (
              <Badge key={r} variant={r === "control-plane" ? "default" : "secondary"}>
                {r}
              </Badge>
            ))}
            <NodeStatusBadge node={n} />
          </div>
        </div>

        <Hardware node={n} />

        <div className="grid gap-3 text-xs">
          <NodeUsage
            icon={<CpuIcon />}
            label="CPU"
            used={n.usage?.cpuMillis}
            requested={n.cpuRequestedMillis}
            capacity={n.cpuCapacityMillis}
            format={formatCores}
          />
          <NodeUsage
            icon={<MemoryStickIcon />}
            label="Memory"
            used={n.usage?.memoryBytes}
            requested={n.memoryRequestedBytes}
            capacity={n.memoryCapacityBytes}
            format={(v) => formatBytes(v)}
          />
        </div>

        <dl className="mt-auto grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 border-t pt-3 text-xs">
          <dt className="text-muted-foreground">OS</dt>
          <dd className="truncate text-right">
            {n.osImage} · {n.architecture}
          </dd>
          <dt className="text-muted-foreground">Kernel</dt>
          <dd className="truncate text-right font-mono">{n.kernelVersion}</dd>
          <dt className="text-muted-foreground">Kubelet</dt>
          <dd className="truncate text-right font-mono">
            {n.kubeletVersion} · {n.containerRuntime.replace("://", " ")}
          </dd>
          <dt className="text-muted-foreground">Pods</dt>
          <dd className="text-right font-mono">
            {n.podsRunning} / {n.podsCapacity}
          </dd>
          <dt className="flex items-center gap-1 text-muted-foreground">
            <HardDriveIcon className="size-3" />
            Ephemeral
          </dt>
          <dd className="text-right font-mono">{formatBytes(n.ephemeralStorageBytes, 0)}</dd>
          <dt className="text-muted-foreground">Joined</dt>
          <dd className="text-right">{formatRelativeTime(n.createdAt)}</dd>
        </dl>
        {n.taints.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {n.taints.map((t) => (
              <span
                key={t}
                className="truncate rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
                title={t}
              >
                {t}
              </span>
            ))}
          </div>
        )}
      </div>
    </GlowCard>
  )
}

function NodeStatusBadge({ node: n }: { node: ClusterNode }) {
  if (!n.ready) return <Badge variant="destructive">{n.status}</Badge>
  if (n.pressure.length > 0)
    return <Badge className="bg-amber-500/15 text-amber-700 dark:text-amber-300">{n.pressure.join(", ")}</Badge>
  if (n.unschedulable) return <Badge variant="outline">Cordoned</Badge>
  return <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">Ready</Badge>
}

function Hardware({ node: n }: { node: ClusterNode }) {
  const hw = n.hardware
  if (hw) {
    return (
      <div className="rounded-xl border bg-muted/30 p-3">
        <div className="flex items-start gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-red-500 to-orange-600 text-white shadow-inner">
            <CpuIcon className="size-4" />
          </div>
          <div className="min-w-0 space-y-0.5">
            <div className="text-sm leading-snug font-medium">{hw.cpuModel || "Unknown CPU"}</div>
            <div className="text-xs text-muted-foreground">
              {hw.cpuCores} cores · {hw.cpuThreads} threads
              {hw.cpuMaxMHz ? ` · up to ${(hw.cpuMaxMHz / 1000).toFixed(2)} GHz` : ""}
              {hw.cpuSockets > 1 ? ` · ${hw.cpuSockets} sockets` : ""}
            </div>
          </div>
        </div>
        {(hw.vendor || hw.product) && (
          <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
            <MonitorIcon className="size-3.5" />
            <span className="truncate">
              {[hw.vendor, hw.product].filter(Boolean).join(" ")}
              {hw.bios ? ` · BIOS ${hw.bios}` : ""}
            </span>
            <Badge variant="outline" className="ml-auto text-[10px]">
              {hw.virtualized ? "virtual machine" : "bare metal"}
            </Badge>
          </div>
        )}
      </div>
    )
  }
  const text = n.hardwareProbing
    ? "Reading hardware…"
    : !n.ready
      ? "Node is not ready, hardware unknown."
      : n.hardwareError
        ? `Hardware probe failed: ${n.hardwareError}`
        : "No hardware data yet."
  return (
    <div className="flex items-center gap-2 rounded-xl border border-dashed p-3 text-xs text-muted-foreground">
      {n.hardwareProbing ? (
        <LoaderCircleIcon className="size-4 animate-spin" />
      ) : (
        <AlertTriangleIcon className="size-4" />
      )}
      {text}
    </div>
  )
}

function NodeUsage({
  icon,
  label,
  used,
  requested,
  capacity,
  format,
}: {
  icon: React.ReactNode
  label: string
  used?: number
  requested: number
  capacity: number
  format: (v: number) => string
}) {
  return (
    <UsageRow
      icon={icon}
      label={label}
      detail={
        <>
          <span className="text-foreground">{used != null ? formatPercent(used, capacity) : "-"}</span> · req{" "}
          {formatPercent(requested, capacity)} · {format(capacity)}
        </>
      }
      value={used}
      requested={requested}
      max={capacity}
    />
  )
}
