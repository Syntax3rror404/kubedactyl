import { useEffect, useState } from "react"
import { TimerIcon } from "lucide-react"

import type { FilesSession } from "@/lib/types"

/**
 * How long the running file container stays without further file operations. The server
 * sends the remaining seconds with every session poll; the countdown runs locally in between.
 */
export function FilesStopCountdown({ session, updatedAt }: { session?: FilesSession; updatedAt: number }) {
  const seconds = session?.stopsInSeconds
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(t)
  }, [])
  if (!session?.ready || seconds === undefined) return null
  const left = Math.max(0, Math.round((updatedAt + seconds * 1000 - now) / 1000))
  return (
    <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <TimerIcon className="size-3.5" />
      {left > 0 ? (
        <>
          The file container stops in{" "}
          <span className="tabular-nums">
            {Math.floor(left / 60)}:{String(left % 60).padStart(2, "0")}
          </span>{" "}
          without file operations.
        </>
      ) : (
        "The file container stops now. Any file operation keeps it running."
      )}
    </p>
  )
}
