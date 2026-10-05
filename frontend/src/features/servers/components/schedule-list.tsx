import { useState } from "react"
import { CalendarClockIcon, PlusIcon, RotateCwIcon, ZapIcon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { EmptyState, QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import { ScheduleCard } from "@/features/servers/components/schedule-card"
import { ScheduleDialog } from "@/features/servers/components/schedule-dialog"
import { dailyRestart, emptySchedule, emptyTask, goodbyeTask } from "@/features/servers/lib/schedules"
import { failed } from "@/lib/notify"
import { useRunSchedule, useSchedules, useUpdateSchedules } from "@/lib/queries"
import type { Schedule, ScheduleView } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

// One state for all dialogs of the page; index null = new entry (index into all schedules).
type Dialog = { kind: "edit"; schedule: Schedule; index: number | null } | { kind: "delete"; name: string } | null

const plain = ({ name, cron, event, enabled, onlyWhenOnline, tasks }: ScheduleView | Schedule): Schedule => ({
  name,
  cron,
  event,
  enabled,
  onlyWhenOnline,
  tasks,
})

// What differs between the Schedules tab (cron) and the Tasks tab (events).
const kinds = {
  schedule: {
    noun: "Schedule",
    intro: "Run console commands and power actions at fixed times.",
    empty: "Restart the server every night, send announcements or run commands regularly.",
    icon: <CalendarClockIcon />,
    create: emptySchedule,
    preset: { label: "Daily restart at 04:00", icon: <RotateCwIcon />, schedule: dailyRestart },
    show: (s: ScheduleView) => !s.event,
  },
  task: {
    noun: "Task",
    intro: "Console commands after a start or before a stop. The stop waits for them.",
    empty: "Greet players after a start, or warn them and save the world before a stop.",
    icon: <ZapIcon />,
    create: emptyTask,
    preset: { label: "Warn and save before stopping", icon: <ZapIcon />, schedule: goodbyeTask },
    show: (s: ScheduleView) => !!s.event,
  },
}

type Kind = keyof typeof kinds

/**
 * The schedules of a server: either the cron based ones (Schedules tab) or the ones that run
 * at events (Tasks tab). Both are saved together as the server's schedule list.
 */
export function ScheduleList({ server, kind }: { server: string; kind: Kind }) {
  const list = useSchedules(server)
  return (
    <QueryState query={list}>
      {(data) => <Schedules server={server} items={data.items} timeZone={data.timeZone} kind={kind} />}
    </QueryState>
  )
}

function Schedules({
  server,
  items,
  timeZone,
  kind,
}: {
  server: string
  items: ScheduleView[]
  timeZone: string
  kind: Kind
}) {
  const k = kinds[kind]
  const [dialog, setDialog] = useState<Dialog>(null)
  const save = useUpdateSchedules(server, { onSuccess: () => setDialog(null) })
  // Saves from the dialog show their error there; toggles and deletes as a toast.
  const saveList = (next: Schedule[], done: string, fromDialog = false) =>
    save.mutate(next, {
      onSuccess: () => toast.success(`${k.noun} ${done}`),
      onError: fromDialog ? undefined : failed(`save the ${k.noun.toLowerCase()}s`),
    })
  const run = useRunSchedule(server, {
    onSuccess: (_, name) => toast.success(`${k.noun} “${name}” started`, { description: "Follow it in the console." }),
    onError: failed(`run the ${k.noun.toLowerCase()}`),
  })
  const editing = dialog?.kind === "edit" ? dialog : null
  const deleting = dialog?.kind === "delete" ? dialog.name : null
  const shown = items.map((s, index) => ({ s, index })).filter(({ s }) => k.show(s))

  const replace = (index: number | null, s: Schedule, fromDialog = false) => {
    const next = items.map(plain)
    if (index === null) next.push(s)
    else next[index] = s
    saveList(next, index === null ? "created" : "saved", fromDialog)
  }
  const remove = (name: string) => saveList(items.filter((s) => s.name !== name).map(plain), "deleted")
  const edit = (schedule: Schedule, index: number | null = null) => {
    save.reset()
    setDialog({ kind: "edit", schedule, index })
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground">
          {k.intro} {kind === "schedule" && `Time zone: ${timeZone}.`}
        </p>
        {shown.length > 0 && (
          <Button size="sm" onClick={() => edit(k.create())}>
            <PlusIcon />
            New {k.noun.toLowerCase()}
          </Button>
        )}
      </div>

      {shown.length === 0 ? (
        <EmptyState icon={k.icon} title={`No ${k.noun.toLowerCase()}s yet`} description={k.empty}>
          <Button variant="outline" onClick={() => edit(k.preset.schedule())}>
            {k.preset.icon}
            {k.preset.label}
          </Button>
          <Button onClick={() => edit(k.create())}>
            <PlusIcon />
            New {k.noun.toLowerCase()}
          </Button>
        </EmptyState>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {shown.map(({ s, index }) => (
            <ScheduleCard
              key={s.name}
              schedule={s}
              timeZone={timeZone}
              busy={save.isPending}
              onToggle={(enabled) => replace(index, { ...plain(s), enabled })}
              onRun={() => run.mutate(s.name)}
              onEdit={() => edit(plain(s), index)}
              onDelete={() => setDialog({ kind: "delete", name: s.name })}
            />
          ))}
        </div>
      )}

      {editing && (
        <ScheduleDialog
          initial={editing.schedule}
          isNew={editing.index === null}
          timeZone={timeZone}
          takenNames={items.filter((_, i) => i !== editing.index).map((s) => s.name)}
          saving={save.isPending}
          errors={fieldErrors(save.error)}
          onSubmit={(s) => replace(editing.index, s, true)}
          onClose={() => (setDialog(null), save.reset())}
        />
      )}

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Delete ${k.noun.toLowerCase()} “${deleting}”?`}
        description={`The ${k.noun.toLowerCase()} stops running. A run in progress finishes its current step.`}
        confirmLabel="Delete"
        destructive
        onConfirm={() => deleting && remove(deleting)}
      />
    </div>
  )
}
