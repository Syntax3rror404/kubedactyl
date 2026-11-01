import { useEffect, useState } from "react"

import type { useConsole } from "@/features/servers/hooks/use-console"

export type Subscribe = ReturnType<typeof useConsole>["subscribe"]

// Lines the user closed during this visit, as "<server> <line>" (oldest dropped first). Kept outside the
// component, so a replayed history (reconnect, another tab and back) does not open them again.
const dismissedLines: string[] = []
const MAX_DISMISSED = 100

/**
 * Watches the console output of a server for any of the patterns (case-insensitive substrings or regular
 * expressions) and returns the last matching line. Every start ("Server marked as starting...", also in the
 * replayed history) clears the match, so only the output of the latest run counts. The one hide rule of all egg
 * feature dialogs: `open` is false while the server runs and for a line the user closed during this visit.
 */
export function useConsoleMatch(server: string, subscribe: Subscribe, patterns: (string | RegExp)[], running: boolean) {
  const [line, setLine] = useState<string | null>(null)
  useEffect(
    () =>
      subscribe(
        (ev) => {
          if (ev.event === "daemon message" && /marked as starting/i.test(ev.args?.[0] ?? "")) return setLine(null)
          if (ev.event !== "console output") return
          const text: string = ev.args?.[0] ?? ""
          const lower = text.toLowerCase()
          const matches = patterns.some((p) => (typeof p === "string" ? lower.includes(p) : p.test(text)))
          if (matches && !dismissedLines.includes(`${server} ${text}`)) setLine(text)
        },
        () => {},
      ),
    // Pattern lists are module constants, so the subscription only changes with the console.
    [server, subscribe, patterns],
  )
  const dismiss = () => {
    if (line === null) return
    dismissedLines.push(`${server} ${line}`)
    if (dismissedLines.length > MAX_DISMISSED) dismissedLines.shift()
    setLine(null)
  }
  return { line, open: line !== null && !running, dismiss }
}
