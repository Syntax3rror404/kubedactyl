import { CalendarClockIcon, PencilIcon, PlayIcon, Trash2Icon, ZapIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { actionLabels, describeCron, eventLabels } from "@/features/servers/lib/schedules"
import { formatRelativeTime } from "@/lib/format"
import type { ScheduleView } from "@/lib/types"
import { cn } from "@/lib/utils"

function resultTone(result?: string) {
  if (!result) return ""
  if (result === "ok") return "text-emerald-600 dark:text-emerald-400"
  if (result.startsWith("skipped")) return "text-amber-600 dark:text-amber-400"
  return "text-red-600 dark:text-red-400"
}

/** One schedule or task: when it runs (cron or event), its steps and the last result. */
export function ScheduleCard({
  schedule: s,
  timeZone,
  busy,
  onToggle,
  onRun,
  onEdit,
  onDelete,
}: {
  schedule: ScheduleView
  timeZone: string
  busy: boolean
  onToggle: (enabled: boolean) => void
  onRun: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const next = s.nextRunAt ? new Date(s.nextRunAt) : null
  return (
    <Card className={cn(!s.enabled && "opacity-70")}>
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <CardTitle className="flex items-center gap-2">
            {s.event ? <ZapIcon className="size-4 shrink-0" /> : <CalendarClockIcon className="size-4 shrink-0" />}
            <span className="truncate">{s.name}</span>
            {s.running && (
              <Badge variant="secondary" className="gap-1">
                <Spinner className="size-3" />
                running
              </Badge>
            )}
          </CardTitle>
          <p className="text-sm text-muted-foreground">
            {s.event ? (
              eventLabels[s.event]
            ) : (
              <>
                {describeCron(s.cron ?? "") ?? s.cron}{" "}
                <span className="font-mono text-xs">
                  ({s.cron}, {timeZone})
                </span>
              </>
            )}
          </p>
        </div>
        <Switch checked={s.enabled} disabled={busy} onCheckedChange={onToggle} aria-label="Enabled" />
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        <ol className="space-y-1.5">
          {s.tasks.map((t, i) => (
            <li key={i} className="flex items-baseline gap-2">
              <span className="w-5 shrink-0 text-right font-mono text-xs text-muted-foreground">{i + 1}.</span>
              <span>
                {t.delaySeconds ? <span className="text-muted-foreground">after {t.delaySeconds}s · </span> : null}
                {actionLabels[t.action]}
                {(t.action === "command" || (t.action === "backup" && t.payload)) && (
                  <code className="ml-1.5 rounded bg-muted px-1.5 py-0.5 font-mono text-xs">{t.payload}</code>
                )}
              </span>
            </li>
          ))}
        </ol>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-xs">
          {!s.event && (
            <>
              <dt className="text-muted-foreground">Next run</dt>
              <dd>{next ? `${next.toLocaleString()}` : s.enabled ? "-" : "disabled"}</dd>
            </>
          )}
          <dt className="text-muted-foreground">Last run</dt>
          <dd>
            {s.lastRunAt ? (
              <>
                {formatRelativeTime(s.lastRunAt)} · <span className={resultTone(s.lastResult)}>{s.lastResult}</span>
              </>
            ) : (
              "never"
            )}
          </dd>
          {s.onlyWhenOnline && (
            <>
              <dt className="text-muted-foreground">Condition</dt>
              <dd>only when the server is online</dd>
            </>
          )}
        </dl>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" disabled={busy || s.running} onClick={onRun}>
            <PlayIcon />
            Run now
          </Button>
          <Button size="sm" variant="outline" disabled={busy} onClick={onEdit}>
            <PencilIcon />
            Edit
          </Button>
          <Button size="sm" variant="ghost" className="text-destructive" disabled={busy} onClick={onDelete}>
            <Trash2Icon />
            Delete
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
