import { useState } from "react"
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useConsoleMatch, type Subscribe } from "@/features/servers/egg-features/use-console-match"
import { failed } from "@/lib/notify"
import { useUpdateAndStartServer } from "@/lib/queries"
import type { Egg, GameServer } from "@/lib/types"

// Pelican's list (Pterodactyl's without the 1.19 line), matched case-insensitively.
const patterns = [
  "java.lang.unsupportedclassversionerror",
  "unsupported major.minor version",
  "has been compiled by a more recent version of the java runtime",
  "minecraft 1.17 requires running the server with java 16 or above",
  "minecraft 1.18 requires running the server with java 17 or above",
  "minecraft 1.19 requires running the server with java 17 or above",
]

/** Egg feature "java_version": the server needs another Java: pick one of the egg's images. */
export function JavaVersionPrompt({
  server,
  egg,
  subscribe,
  running,
  isAdmin,
}: {
  server: GameServer
  egg: Egg
  subscribe: Subscribe
  running: boolean
  isAdmin: boolean
}) {
  const { line, dismiss } = useConsoleMatch(subscribe, patterns)
  const name = server.metadata.name
  const images = egg.spec.dockerImages.filter((i) => i.image !== server.spec.image)
  // Like Pelican: a custom image set by an administrator is not replaced by users.
  const custom = !egg.spec.dockerImages.some((i) => i.image === server.spec.image)
  const canChange = images.length > 0 && (isAdmin || !custom)
  const [image, setImage] = useState("")
  const picked = image || images[0]?.image || ""
  const update = useUpdateAndStartServer(name, {
    onSuccess: () => {
      dismiss()
      toast.success("Docker image updated", { description: "Starting the server…" })
    },
    onError: failed("update the docker image"),
  })
  return (
    <AlertDialog open={!!line && !running} onOpenChange={(o) => !o && dismiss()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Unsupported Java version</AlertDialogTitle>
          <AlertDialogDescription>
            This Java version is not supported, so the server cannot start.
            {canChange
              ? " Pick a supported version to continue."
              : custom
                ? " Ask an administrator to change the custom image."
                : ""}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {canChange && (
          <Select value={picked} onValueChange={setImage}>
            <SelectTrigger className="w-full" aria-label="Docker image">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {images.map((i) => (
                <SelectItem key={i.image} value={i.image}>
                  {i.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          {canChange && (
            <AlertDialogAction
              disabled={update.isPending}
              onClick={(e) => {
                e.preventDefault()
                update.mutate({ changes: { image: picked }, signal: "start" })
              }}
            >
              Update docker image
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
