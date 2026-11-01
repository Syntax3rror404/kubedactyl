import { toast } from "sonner"

import { FeaturePrompt } from "@/features/servers/egg-features/feature-prompt"
import { useConsoleMatch, type Subscribe } from "@/features/servers/egg-features/use-console-match"
import { failed } from "@/lib/notify"
import { useAcceptEula } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

const patterns = ["you need to agree to the eula in order to run the server"]

/**
 * Egg feature "eula": offers to accept the Minecraft EULA (writes eula.txt and starts the server). Accepting waits
 * until the server has stopped, a start before would be lost.
 */
export function EulaPrompt({
  server,
  subscribe,
  active,
  running,
}: {
  server: GameServer
  subscribe: Subscribe
  active: boolean
  running: boolean
}) {
  const { open, dismiss } = useConsoleMatch(server.metadata.name, subscribe, patterns, running)
  const accept = useAcceptEula(server.metadata.name, {
    onSuccess: () => {
      dismiss()
      toast.success("EULA accepted", { description: "Starting the server…" })
    },
    onError: failed("accept the EULA"),
  })

  return (
    <FeaturePrompt
      open={open}
      onClose={dismiss}
      title="Accept the Minecraft EULA"
      description={
        <>
          By pressing “I accept” you confirm that you have read and agree to the{" "}
          <a
            className="underline underline-offset-4"
            href="https://aka.ms/MinecraftEULA"
            target="_blank"
            rel="noreferrer"
          >
            Minecraft EULA
          </a>
          . This writes <code>eula=true</code> to <code>eula.txt</code> and starts the server.
        </>
      }
      confirm={{ label: "I accept", pending: accept.isPending || active, onConfirm: () => accept.mutate() }}
    />
  )
}
