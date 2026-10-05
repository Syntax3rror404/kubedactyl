import { useOutletContext } from "react-router"

import { ScheduleList } from "@/features/servers/components/schedule-list"
import type { ServerContext } from "@/features/servers/server-layout"

/**
 * /servers/:server/tasks: console commands after the server started or before it stops. They are
 * schedules with an event instead of a cron expression (saved together with the schedules).
 */
export function TasksPage() {
  const { server } = useOutletContext<ServerContext>()
  return <ScheduleList server={server.metadata.name} kind="task" />
}
