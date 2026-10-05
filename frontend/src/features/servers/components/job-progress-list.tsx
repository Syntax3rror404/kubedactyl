import { ProgressStripe } from "@/components/common/callout"
import { JobStateIcon } from "@/components/common/job-state-icon"
import { formatDuration, formatRelativeTime, secondsSince } from "@/lib/format"
import type { ServerJob } from "@/lib/types"

const verbs: Record<ServerJob["kind"], string> = {
  backup: "Creating backup",
  restore: "Restoring",
  pull: "Downloading",
}

/** Running and recently finished background jobs (backups, restores, downloads). */
export function JobProgressList({
  jobs,
  kinds,
  limit = 5,
}: {
  jobs?: ServerJob[]
  kinds: ServerJob["kind"][]
  limit?: number
}) {
  // Finished jobs stay visible for ten minutes.
  const recent = (jobs ?? [])
    .filter((j) => kinds.includes(j.kind) && (j.state === "running" || secondsSince(j.finishedAt ?? j.startedAt) < 600))
    .slice(0, limit)
  if (recent.length === 0) return null
  return (
    <ul className="divide-y rounded-xl border bg-card text-sm">
      {recent.map((j) => (
        <li key={j.id} className="relative flex items-center gap-3 overflow-hidden px-4 py-2.5">
          {j.state === "running" && <ProgressStripe />}
          <JobStateIcon state={j.state} />
          <div className="min-w-0 flex-1">
            <div className="truncate">
              {verbs[j.kind]} <span className="font-mono text-xs">{j.label}</span>
            </div>
            {j.error && <div className="truncate text-xs text-destructive">{j.error}</div>}
          </div>
          <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
            {j.state === "running" ? formatDuration(secondsSince(j.startedAt)) : formatRelativeTime(j.finishedAt)}
          </span>
        </li>
      ))}
    </ul>
  )
}
