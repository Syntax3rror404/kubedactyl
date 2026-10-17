import { ExternalLinkIcon } from "lucide-react"

import { AlertDialogAction } from "@/components/ui/alert-dialog"
import { FeaturePrompt } from "@/features/servers/egg-features/feature-prompt"
import { useConsoleMatch, type Subscribe } from "@/features/servers/egg-features/use-console-match"

// The device login URL printed by the Hytale downloader.
const hytalePattern = [/https:\/\/oauth\.accounts\.hytale\.com\/oauth2\/device\/verify\?user_code=\S+/i]

/** Egg feature "hytale_oauth": the Hytale downloader waits for a device login. */
export function HytaleLoginPrompt({ subscribe, running }: { subscribe: Subscribe; running: boolean }) {
  const { line, dismiss } = useConsoleMatch(subscribe, hytalePattern)
  const url = line?.match(hytalePattern[0])?.[0]
  return (
    <FeaturePrompt
      open={!!url && !running}
      onClose={dismiss}
      title="Authentication required"
      description="Log in with your Hytale account so the server can download or update its files."
      action={
        <AlertDialogAction asChild>
          <a href={url} target="_blank" rel="noopener noreferrer" onClick={dismiss}>
            <ExternalLinkIcon />
            Log in
          </a>
        </AlertDialogAction>
      }
    />
  )
}
