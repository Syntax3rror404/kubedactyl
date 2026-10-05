import { PlusIcon, XIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { actionLabels, payloadInput } from "@/features/servers/lib/schedules"
import type { ScheduleAction, ScheduleTask } from "@/lib/types"

const MAX_TASKS = 10

/** The steps of a schedule: action (tasks only run commands), payload and delay, in order. */
export function ScheduleSteps({
  tasks,
  isTask,
  onChange,
}: {
  tasks: ScheduleTask[]
  isTask: boolean
  onChange: (tasks: ScheduleTask[]) => void
}) {
  const setTask = (i: number, patch: Partial<ScheduleTask>) =>
    onChange(tasks.map((t, j) => (j === i ? { ...t, ...patch } : t)))
  return (
    <div className="space-y-2">
      <FieldLabel>Steps</FieldLabel>
      {tasks.map((t, i) => (
        <div
          key={i}
          className="grid grid-cols-[1.5rem_1fr_auto] items-start gap-2 rounded-lg border bg-muted/30 p-2 sm:grid-cols-[1.5rem_11rem_1fr_7rem_auto]"
        >
          <span className="pt-2 text-right font-mono text-xs text-muted-foreground">{i + 1}.</span>
          {isTask ? (
            <span className="pt-2 text-sm">{actionLabels.command}</span>
          ) : (
            <Select
              value={t.action}
              onValueChange={(action) => setTask(i, { action: action as ScheduleAction, payload: "" })}
            >
              <SelectTrigger className="w-full" aria-label="Action">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(Object.keys(actionLabels) as ScheduleAction[]).map((a) => (
                  <SelectItem key={a} value={a}>
                    {actionLabels[a]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <Input
            className="col-span-2 font-mono sm:col-span-1"
            disabled={!payloadInput[t.action]}
            value={payloadInput[t.action] ? (t.payload ?? "") : ""}
            onChange={(e) => setTask(i, { payload: e.target.value })}
            placeholder={payloadInput[t.action]?.placeholder ?? "-"}
            aria-label={payloadInput[t.action]?.label ?? "Payload"}
            maxLength={payloadInput[t.action]?.maxLength ?? 500}
          />
          <Input
            type="number"
            min={0}
            max={900}
            value={t.delaySeconds ?? 0}
            onChange={(e) => setTask(i, { delaySeconds: Math.min(900, Math.max(0, Number(e.target.value) || 0)) })}
            aria-label="Delay in seconds"
            title="Seconds to wait before this task"
          />
          <Button
            type="button"
            size="icon"
            variant="ghost"
            disabled={tasks.length === 1}
            onClick={() => onChange(tasks.filter((_, j) => j !== i))}
            aria-label="Remove step"
          >
            <XIcon />
          </Button>
        </div>
      ))}
      <div className="flex items-center justify-between">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={tasks.length >= MAX_TASKS}
          onClick={() => onChange([...tasks, { action: "command", payload: "", delaySeconds: 0 }])}
        >
          <PlusIcon />
          Add step
        </Button>
        <span className="text-xs text-muted-foreground">Delay = seconds to wait before the step (max. 900)</span>
      </div>
    </div>
  )
}
