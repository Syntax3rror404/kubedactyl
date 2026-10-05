import { useEffect, useEffectEvent, useRef } from "react"
import { toast } from "sonner"

import { useFilesSession, useRefreshFiles, useServerJobs } from "@/lib/queries"
import type { ServerJob } from "@/lib/types"

const kindLabels: Record<ServerJob["kind"], string> = { backup: "Backup", restore: "Restore", pull: "Download" }

/**
 * The file container of a server for the pages that need it (files, backups): starts it on demand
 * (see useFilesSession) and follows the background jobs. Jobs that finish while the page is open are
 * announced with a toast and reload the listings and backups.
 */
export function useFileContainer(server: string) {
  const files = useFilesSession(server)
  const refresh = useRefreshFiles(server)
  const jobs = useServerJobs(server)
  const running = useRef<Set<string> | null>(null)
  const announce = useEffectEvent((j: ServerJob) => {
    if (j.state === "done") toast.success(`${kindLabels[j.kind]} finished`, { description: j.label })
    else toast.error(`${kindLabels[j.kind]} failed`, { description: j.error || j.label })
    refresh()
  })

  useEffect(() => {
    const items = jobs.data
    if (!items) return
    const before = running.current
    running.current = new Set(items.filter((j) => j.state === "running").map((j) => j.id))
    if (!before) return
    for (const j of items) {
      if (before.has(j.id) && j.state !== "running") announce(j)
    }
  }, [jobs.data])

  return { files, ready: files.view === "ready", refresh, jobs }
}
