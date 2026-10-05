import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useConsoleMatch, type Subscribe } from "@/features/servers/egg-features/use-console-match"
import { failed } from "@/lib/notify"
import { useAcceptEula } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

const patterns = ["you need to agree to the eula in order to run the server"]

/**
 * Pterodactyl egg feature "eula": when the server asks for the Minecraft EULA, offer to
 * accept it (writes eula.txt and starts the server). The console history can still hold
 * the line of an earlier run, so the prompt is not shown while the server is running.
 */
export function EulaPrompt({
  server,
  subscribe,
  active,
}: {
  server: GameServer
  subscribe: Subscribe
  active: boolean
}) {
  const { line, dismiss } = useConsoleMatch(subscribe, patterns)
  const accept = useAcceptEula(server.metadata.name, {
    onSuccess: () => {
      dismiss()
      toast.success("EULA accepted", { description: "Starting the server…" })
    },
    onError: failed("accept the EULA"),
  })

  return (
    <AlertDialog open={!!line && !active} onOpenChange={(o) => !o && dismiss()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Accept the Minecraft EULA</AlertDialogTitle>
          <AlertDialogDescription>
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
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={accept.isPending}
            onClick={(e) => {
              e.preventDefault()
              accept.mutate()
            }}
          >
            I accept
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
