import { XIcon } from "lucide-react"

import { ProgressStripe } from "@/components/common/callout"
import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { JobStateIcon } from "@/components/common/job-state-icon"
import { ProgressBar } from "@/components/common/progress-bar"
import { Button } from "@/components/ui/button"
import { jobKinds } from "@/features/servers/lib/jobs"
import { formatBytes, formatDuration, formatRelativeTime, secondsSince } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useCancelJob } from "@/lib/queries"
import type { ServerJob } from "@/lib/types"
import { cn } from "@/lib/utils"

type Progress = NonNullable<ServerJob["progress"]>

/** An amount in the unit of the progress: "1,204 files" or "1.2 GiB" (bytes of an archive read or a download). */
function amount(unit: Progress["unit"], n: number) {
  return unit === "bytes" ? formatBytes(n) : `${Math.round(n).toLocaleString("en-US")} files`
}

/**
 * Size, speed and time left of a running job, from the average since its first progress report (steady, needs no
 * history). Speed and time left show after a few seconds of measuring; without a total (a download whose server
 * sends no size) only what is done and the speed.
 */
function transfer(p: Progress) {
  const seconds = secondsSince(p.startedAt)
  const rate = seconds >= 3 ? p.done / seconds : 0
  const known = p.total > 0
  const done = p.unit === "bytes" ? formatBytes(p.done) : p.done.toLocaleString("en-US")
  return {
    size: known ? `${done} / ${amount(p.unit, p.total)}` : amount(p.unit, p.done),
    speed: rate > 0 ? `${amount(p.unit, rate)}/s` : "",
    left: known && rate > 0 ? `${formatDuration(Math.max(1, (p.total - p.done) / rate))} left` : "",
  }
}

/**
 * Running and recently finished background jobs (backups, restores, downloads, archives) of a server. A running job
 * shows a bar (sliding while its end is not known), a button that cancels it and, once it reports its progress,
 * size, speed, time left and, for backups and restores, the file it works on.
 */
export function JobProgressList({
  server,
  jobs,
  kinds: shown,
  limit = 5,
  showDone = true,
}: {
  server: string
  jobs?: ServerJob[]
  kinds: ServerJob["kind"][]
  limit?: number
  // false hides a job as soon as it has finished well or was cancelled (a toast says so); failed ones stay.
  showDone?: boolean
}) {
  // Finished jobs stay visible for ten minutes.
  const recent = (jobs ?? [])
    .filter((j) => shown.includes(j.kind) && (showDone || j.state === "running" || j.state === "failed"))
    .filter((j) => j.state === "running" || secondsSince(j.finishedAt ?? j.startedAt) < 600)
    .slice(0, limit)
  if (recent.length === 0) return null
  return (
    <ul className="divide-y rounded-xl border bg-card text-sm">
      {recent.map((j) => (
        <JobRow key={j.id} server={server} job={j} />
      ))}
    </ul>
  )
}

function JobRow({ server, job: j }: { server: string; job: ServerJob }) {
  const { verb, icon: Icon } = jobKinds[j.kind]
  const p = j.state === "running" ? j.progress : undefined
  const t = p && transfer(p)
  return (
    <li className="relative space-y-2 overflow-hidden px-4 py-3">
      {j.state === "running" && <ProgressStripe />}
      <div className="flex items-center gap-2">
        <JobStateIcon state={j.state} />
        <span className="min-w-0 flex-1 truncate">
          {verb} <span className="font-mono text-xs">{j.label}</span>
        </span>
        <span className={cn("shrink-0 text-xs text-muted-foreground tabular-nums", t && "font-mono")}>
          {t
            ? t.size
            : j.state === "running"
              ? formatDuration(secondsSince(j.startedAt))
              : formatRelativeTime(j.finishedAt)}
        </span>
        {j.state === "running" && <CancelButton server={server} job={j} />}
      </div>
      {j.state === "running" && <ProgressBar value={p?.total ? p.done : undefined} max={p?.total} className="h-1.5" />}
      {t && p && (
        <div className="flex items-center gap-3 font-mono text-xs text-muted-foreground tabular-nums">
          <Icon className="size-3.5 shrink-0 text-sky-500" />
          <span className="shrink-0">{t.speed}</span>
          <span className="min-w-0 flex-1 truncate text-center" title={p.file}>
            {p.file}
          </span>
          <span className="shrink-0">{t.left}</span>
        </div>
      )}
      {j.error && <div className="truncate text-xs text-destructive">{j.error}</div>}
    </li>
  )
}

/** Cancels a job; a restore asks first, because it has already deleted the server files. */
function CancelButton({ server, job }: { server: string; job: ServerJob }) {
  const cancel = useCancelJob(server, { onError: failed("cancel the job") })
  const restore = job.kind === "restore"
  const button = (
    <Button
      variant="ghost"
      size="icon-xs"
      aria-label="Cancel"
      title="Cancel"
      disabled={cancel.isPending}
      onClick={restore ? undefined : () => cancel.mutate(job.id)}
    >
      <XIcon />
    </Button>
  )
  if (!restore) return button
  return (
    <ConfirmDialog
      trigger={button}
      title="Cancel the restore?"
      description="The server files stay incomplete until a backup is restored again."
      confirmLabel="Cancel restore"
      cancelLabel="Keep restoring"
      destructive
      onConfirm={() => cancel.mutate(job.id)}
    />
  )
}
