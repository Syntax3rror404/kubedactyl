import { PlayIcon, RotateCwIcon, SkullIcon, SquareIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { powerFeedback } from "@/features/servers/lib/feedback"
import { activePhases, phaseOf, stoppedPhases } from "@/lib/format"
import { useSendPower } from "@/lib/queries"
import type { GameServer, PowerSignal } from "@/lib/types"

/** Start, restart, stop and kill buttons (enabled according to the phase and the suspension). */
export function PowerControls({ server, size = "default" }: { server: GameServer; size?: "default" | "sm" }) {
  const power = useSendPower(server.metadata.name, powerFeedback)
  const phase = phaseOf(server)
  const busy = power.isPending
  const running = phase === "Running" || phase === "Starting"
  const suspended = !!server.spec.suspended
  const canStart = !suspended && stoppedPhases.includes(phase)
  const canStop = running
  const canKill = activePhases.includes(phase)

  const btn = (
    signal: PowerSignal,
    label: string,
    icon: React.ReactNode,
    enabled: boolean,
    variant: "default" | "outline" | "destructive" | "secondary",
  ) => (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            size={size === "sm" ? "icon-sm" : "sm"}
            variant={variant}
            disabled={!enabled || busy}
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              power.mutate(signal)
            }}
          >
            {icon}
            {size !== "sm" && <span>{label}</span>}
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )

  return (
    <div className="flex items-center gap-1.5">
      {btn("start", "Start", <PlayIcon />, canStart, "default")}
      {btn("restart", "Restart", <RotateCwIcon />, running && !suspended, "outline")}
      {btn("stop", "Stop", <SquareIcon />, canStop, "outline")}
      {btn("kill", "Kill", <SkullIcon />, canKill, "destructive")}
    </div>
  )
}
