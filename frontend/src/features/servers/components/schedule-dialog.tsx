import { SaveIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { ScheduleSteps } from "@/features/servers/components/schedule-steps"
import { cronPresets, describeCron, eventLabels, MAX_STOP_DELAY } from "@/features/servers/lib/schedules"
import { useDraft } from "@/hooks/use-draft"
import type { Schedule, ScheduleEvent } from "@/lib/types"

/**
 * Create or edit a schedule (name, cron expression with presets, condition, steps) or a task: a
 * schedule with an event instead of a cron expression, whose steps are console commands only.
 */
export function ScheduleDialog({
  initial,
  isNew,
  timeZone,
  takenNames,
  saving,
  errors,
  onSubmit,
  onClose,
}: {
  initial: Schedule
  isNew: boolean
  timeZone: string
  takenNames: string[]
  saving: boolean
  errors: Record<string, string>
  onSubmit: (s: Schedule) => void
  onClose: () => void
}) {
  const { draft: s, set } = useDraft<Schedule>(initial)
  const isTask = !!initial.event
  const noun = isTask ? "task" : "schedule"
  const description = isTask ? "" : describeCron(s.cron ?? "")
  const nameTaken = takenNames.some((n) => n.toLowerCase() === s.name.trim().toLowerCase())
  const tasksValid = s.tasks.length > 0 && s.tasks.every((t) => t.action !== "command" || t.payload?.trim())
  const stopDelay = s.event === "stopping" ? s.tasks.reduce((sum, t) => sum + (t.delaySeconds ?? 0), 0) : 0
  const tooSlow = stopDelay > MAX_STOP_DELAY
  const valid = s.name.trim() !== "" && !nameTaken && description !== null && tasksValid && !tooSlow

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!valid) return
    onSubmit(
      isTask ? { ...s, name: s.name.trim(), cron: undefined } : { ...s, name: s.name.trim(), cron: s.cron?.trim() },
    )
  }

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <form onSubmit={submit} className="contents">
          <DialogHeader>
            <DialogTitle>{isNew ? `New ${noun}` : `Edit ${initial.name}`}</DialogTitle>
            <DialogDescription>
              {isTask
                ? "The steps run one after another. Before a stop, the server stays online until they are done."
                : `The steps run one after another. Times are in the panel time zone (${timeZone}).`}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5">
            <Field data-invalid={nameTaken}>
              <FieldLabel htmlFor="schedule-name">Name</FieldLabel>
              <Input
                id="schedule-name"
                value={s.name}
                maxLength={50}
                onChange={(e) => set("name", e.target.value)}
                placeholder="Daily restart"
              />
              {nameTaken && <FieldError>A schedule or task with this name exists already.</FieldError>}
            </Field>

            {isTask ? (
              <Field>
                <FieldLabel htmlFor="task-event">When</FieldLabel>
                <Select value={s.event} onValueChange={(event) => set("event", event as ScheduleEvent)}>
                  <SelectTrigger id="task-event" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(Object.keys(eventLabels) as ScheduleEvent[]).map((e) => (
                      <SelectItem key={e} value={e}>
                        {eventLabels[e]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FieldDescription>
                  {s.event === "stopping"
                    ? `Runs on stop and restart (not on kill or a crash); the delays may add up to ${MAX_STOP_DELAY} seconds.`
                    : "Runs once the server is marked as running (its startup line appeared)."}
                </FieldDescription>
              </Field>
            ) : (
              <Field data-invalid={description === null}>
                <FieldLabel htmlFor="schedule-cron">When</FieldLabel>
                <div className="flex flex-wrap gap-1.5">
                  {cronPresets.map((p) => (
                    <Button
                      key={p.cron}
                      type="button"
                      size="sm"
                      variant={s.cron === p.cron ? "secondary" : "outline"}
                      className="h-7 text-xs"
                      onClick={() => set("cron", p.cron)}
                    >
                      {p.label}
                    </Button>
                  ))}
                </div>
                <Input
                  id="schedule-cron"
                  className="font-mono"
                  value={s.cron}
                  onChange={(e) => set("cron", e.target.value)}
                  placeholder="0 4 * * *"
                />
                {description === null ? (
                  <FieldError>
                    Not a valid cron expression (minute hour day month weekday, or @daily, @hourly, …).
                  </FieldError>
                ) : (
                  <FieldDescription>{description}</FieldDescription>
                )}
              </Field>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <Field orientation="horizontal">
                <Switch
                  id="schedule-enabled"
                  checked={s.enabled}
                  onCheckedChange={(enabled) => set("enabled", enabled)}
                />
                <FieldLabel htmlFor="schedule-enabled">Enabled</FieldLabel>
              </Field>
              {!isTask && (
                <Field orientation="horizontal">
                  <Switch
                    id="schedule-online"
                    checked={s.onlyWhenOnline ?? false}
                    onCheckedChange={(onlyWhenOnline) => set("onlyWhenOnline", onlyWhenOnline)}
                  />
                  <FieldLabel htmlFor="schedule-online">Only when the server is online</FieldLabel>
                </Field>
              )}
            </div>

            <ScheduleSteps tasks={s.tasks} isTask={isTask} onChange={(tasks) => set("tasks", tasks)} />
            {tooSlow && (
              <FieldError>
                The delays add up to {stopDelay} seconds, at most {MAX_STOP_DELAY} before a stop.
              </FieldError>
            )}
            {errors.form && <FieldError>{errors.form}</FieldError>}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={!valid || saving}>
              {saving ? <Spinner /> : <SaveIcon />}
              Save {noun}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
