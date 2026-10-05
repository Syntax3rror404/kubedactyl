import { RefreshCwIcon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { powerFeedback } from "@/features/servers/lib/feedback"
import { activePhases, phaseOf } from "@/lib/format"
import { useSendPower } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Shown while the running server still uses older runtime settings (status.restartRequired). */
export function RestartBanner({ server }: { server: GameServer }) {
  const power = useSendPower(server.metadata.name, powerFeedback)
  if (!server.status?.restartRequired || !activePhases.includes(phaseOf(server))) return null
  return (
    <Callout
      tone="info"
      icon={<RefreshCwIcon />}
      action={
        <Button size="sm" disabled={power.isPending} onClick={() => power.mutate("restart")}>
          {power.isPending ? <Spinner /> : <RefreshCwIcon />}
          Restart now
        </Button>
      }
    >
      <p>
        <span className="font-medium">Restart required.</span> The settings changed while the server was running; they
        apply after a restart.
      </p>
    </Callout>
  )
}
