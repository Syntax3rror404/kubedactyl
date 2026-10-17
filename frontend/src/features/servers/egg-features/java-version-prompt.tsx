import { useState } from "react"
import { toast } from "sonner"

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { FeaturePrompt } from "@/features/servers/egg-features/feature-prompt"
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

/** Egg feature "java_version": the server needs another Java, so the user picks one of the egg's images. */
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
  const images = egg.spec.dockerImages.filter((i) => i.image !== server.spec.image)
  // Users never replace a custom image an administrator has set.
  const custom = !egg.spec.dockerImages.some((i) => i.image === server.spec.image)
  const canChange = images.length > 0 && (isAdmin || !custom)
  const [image, setImage] = useState("")
  const picked = image || images[0]?.image || ""
  const update = useUpdateAndStartServer(server.metadata.name, {
    onSuccess: () => {
      dismiss()
      toast.success("Docker image updated", { description: "Starting the server…" })
    },
    onError: failed("update the docker image"),
  })
  const hint = canChange
    ? " Pick a supported version to continue."
    : custom
      ? " Ask an administrator to change the custom image."
      : ""
  return (
    <FeaturePrompt
      open={!!line && !running}
      onClose={dismiss}
      title="Unsupported Java version"
      description={`This Java version is not supported, so the server cannot start.${hint}`}
      confirm={
        canChange
          ? {
              label: "Update docker image",
              pending: update.isPending,
              onConfirm: () => update.mutate({ changes: { image: picked }, signal: "start" }),
            }
          : undefined
      }
    >
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
    </FeaturePrompt>
  )
}
