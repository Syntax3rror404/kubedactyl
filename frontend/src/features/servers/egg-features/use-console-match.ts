import { useEffect, useState } from "react"

import type { useConsole } from "@/features/servers/hooks/use-console"

export type Subscribe = ReturnType<typeof useConsole>["subscribe"]

/**
 * Watches the console output for any of the patterns (case-insensitive substrings like
 * Pterodactyl, or regular expressions) and returns the last matching line until dismissed.
 * Every start ("Server marked as starting...", also in the replayed history) clears the match,
 * so only the output of the latest run counts.
 */
export function useConsoleMatch(subscribe: Subscribe, patterns: (string | RegExp)[]) {
  const [line, setLine] = useState<string | null>(null)
  useEffect(
    () =>
      subscribe(
        (ev) => {
          if (ev.event === "daemon message" && /marked as starting/i.test(ev.args?.[0] ?? "")) return setLine(null)
          if (ev.event !== "console output") return
          const text: string = ev.args?.[0] ?? ""
          const lower = text.toLowerCase()
          if (patterns.some((p) => (typeof p === "string" ? lower.includes(p) : p.test(text)))) setLine(text)
        },
        () => {},
      ),
    // Pattern lists are module constants, so the subscription only changes with the console.
    [subscribe, patterns],
  )
  return { line, dismiss: () => setLine(null) }
}
