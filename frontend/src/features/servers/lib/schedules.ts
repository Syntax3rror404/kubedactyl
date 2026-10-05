import cronstrue from "cronstrue"

import type { Schedule, ScheduleAction, ScheduleEvent } from "@/lib/types"

/** Display names of the schedule task actions. */
export const actionLabels: Record<ScheduleAction, string> = {
  command: "Send command",
  start: "Start server",
  stop: "Stop server",
  restart: "Restart server",
  kill: "Kill server",
  backup: "Create backup",
}

/** Actions whose payload input is used: the console command or the backup label. */
export const payloadInput: Partial<Record<ScheduleAction, { placeholder: string; label: string; maxLength: number }>> =
  {
    command: { placeholder: "say Restart in 5 minutes", label: "Command", maxLength: 500 },
    backup: { placeholder: "Label (optional)", label: "Backup label", maxLength: 40 },
  }

/** Common cron expressions offered as one click presets. */
export const cronPresets: { label: string; cron: string }[] = [
  { label: "Every day at 04:00", cron: "0 4 * * *" },
  { label: "Every 6 hours", cron: "0 */6 * * *" },
  { label: "Every hour", cron: "0 * * * *" },
  { label: "Every 15 minutes", cron: "*/15 * * * *" },
  { label: "Mondays at 06:00", cron: "0 6 * * 1" },
]

/** Human readable cron expression, or null when it is invalid. */
export function describeCron(expr: string): string | null {
  const e = expr.trim()
  if (!e || e.startsWith("@every") || /^(CRON_)?TZ=/i.test(e)) return null
  try {
    return cronstrue.toString(e, { use24HourTimeFormat: true })
  } catch {
    return null
  }
}

/** Starting point for new schedules: warn the players, then restart. */
export const dailyRestart = (): Schedule => ({
  name: "Daily restart",
  cron: "0 4 * * *",
  enabled: true,
  onlyWhenOnline: true,
  tasks: [
    { action: "command", payload: "say Server restarts in 5 minutes" },
    { action: "restart", delaySeconds: 300 },
  ],
})

/** A new schedule: daily at 04:00 with one command task. */
export const emptySchedule = (): Schedule => ({
  name: "",
  cron: "0 4 * * *",
  enabled: true,
  onlyWhenOnline: false,
  tasks: [{ action: "command", payload: "" }],
})

/** When a task (a schedule with an event) runs. */
export const eventLabels: Record<ScheduleEvent, string> = {
  started: "After the server started",
  stopping: "Before the server stops or restarts",
}

/** The delays of "stopping" tasks may add up to this many seconds (the stop waits for them). */
export const MAX_STOP_DELAY = 300

/** A new task: a command right after the server started. */
export const emptyTask = (): Schedule => ({
  name: "",
  event: "started",
  enabled: true,
  tasks: [{ action: "command", payload: "" }],
})

/** Starting point for stop tasks: warn the players, wait, save. */
export const goodbyeTask = (): Schedule => ({
  name: "Warn and save",
  event: "stopping",
  enabled: true,
  tasks: [
    { action: "command", payload: "say The server stops in 30 seconds" },
    { action: "command", payload: "save-all", delaySeconds: 30 },
  ],
})
