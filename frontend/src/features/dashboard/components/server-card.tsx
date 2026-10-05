import { CpuIcon, HardDriveIcon, MemoryStickIcon, UserIcon } from "lucide-react"
import { useNavigate } from "react-router"

import { CopyButton } from "@/components/common/copy-button"
import { EggIcon } from "@/components/common/egg-icon"
import { GlowCard } from "@/components/common/glow-card"
import { StatusBadge, SuspendedBadge } from "@/components/common/status-badge"
import { UsageRow } from "@/components/common/usage-row"
import { PowerControls } from "@/features/servers/components/power-controls"
import { useAuth } from "@/hooks/use-auth"
import {
  activePhases,
  formatBytes,
  formatCpuPercent,
  formatMiB,
  phaseOf,
  serverAddress,
  serverEggName,
  serverName,
  serverOwner,
} from "@/lib/format"
import { useSettings, useServerStats } from "@/lib/queries"
import type { Egg, GameServer } from "@/lib/types"

/** Dashboard card of a server with live usage and power buttons (glowing border on hover). */
export function ServerCard({ server, egg }: { server: GameServer; egg?: Egg }) {
  const navigate = useNavigate()
  const phase = phaseOf(server)
  const active = activePhases.includes(phase)
  const stats = useServerStats(server.metadata.name, active)
  const address = serverAddress(server, useSettings().data?.externalDomain)
  const r = server.spec.resources
  const suspended = !!server.spec.suspended
  const { isAdmin } = useAuth()
  const eggName = serverEggName(server, egg)

  return (
    <div
      role="link"
      tabIndex={0}
      onClick={() => navigate(`/servers/${server.metadata.name}`)}
      onKeyDown={(e) => e.key === "Enter" && navigate(`/servers/${server.metadata.name}`)}
      className="h-full cursor-pointer rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <GlowCard>
        <div className="flex h-full flex-col gap-5 p-5">
          <div className="flex items-start gap-3">
            <EggIcon name={eggName} icon={egg?.spec.icon} />
            <div className="min-w-0 flex-1">
              <h3 className="truncate font-semibold tracking-tight">{serverName(server)}</h3>
              <p className="flex items-center gap-1.5 truncate text-xs text-muted-foreground">
                {eggName}
                {isAdmin && (
                  <span className="inline-flex items-center gap-1 rounded-md bg-muted px-1.5 py-0.5 font-mono text-[10px]">
                    <UserIcon className="size-3" />
                    {serverOwner(server) ?? "system"}
                  </span>
                )}
              </p>
            </div>
            {suspended ? <SuspendedBadge /> : <StatusBadge phase={phase} />}
          </div>

          <div className="flex items-center justify-between rounded-lg border bg-muted/40 px-3 py-1.5 font-mono text-sm">
            <span className={address ? "" : "text-muted-foreground"}>{address ?? "Waiting for IP…"}</span>
            {address && <CopyButton value={address} label="Copy address" />}
          </div>

          <div className="grid gap-3 text-xs">
            <UsageRow
              icon={<CpuIcon />}
              label="CPU"
              detail={
                <>
                  <span className="text-foreground">{formatCpuPercent(stats.data?.cpuMillis)}</span> /{" "}
                  {r.cpuMillis ? formatCpuPercent(r.cpuMillis) : "∞"}
                </>
              }
              value={stats.data?.cpuMillis}
              max={r.cpuMillis || null}
            />
            <UsageRow
              icon={<MemoryStickIcon />}
              label="Memory"
              detail={
                <>
                  <span className="text-foreground">{formatBytes(active ? stats.data?.memoryBytes : null)}</span> /{" "}
                  {formatMiB(r.memoryMiB)}
                </>
              }
              value={active ? stats.data?.memoryBytes : null}
              max={r.memoryMiB * 1024 * 1024}
            />
            <UsageRow
              icon={<HardDriveIcon />}
              label="Disk"
              detail={
                <>
                  <span className="text-foreground">{formatBytes(stats.data?.diskBytes)}</span> / {formatMiB(r.diskMiB)}
                </>
              }
              value={stats.data?.diskBytes}
              max={r.diskMiB * 1024 * 1024}
            />
          </div>

          <div className="mt-auto flex items-center justify-between border-t pt-4">
            <span className="truncate text-xs text-muted-foreground">
              {server.status?.message || server.spec.image.split("/").pop()}
            </span>
            {(!suspended || isAdmin) && <PowerControls server={server} size="sm" />}
          </div>
        </div>
      </GlowCard>
    </div>
  )
}
