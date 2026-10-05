import { useOutletContext } from "react-router"

import { ScheduleList } from "@/features/servers/components/schedule-list"
import type { ServerContext } from "@/features/servers/server-layout"

/** /servers/:server/schedules: commands and power actions at fixed times. */
export function SchedulesPage() {
  const { server } = useOutletContext<ServerContext>()
  return <ScheduleList server={server.metadata.name} kind="schedule" />
}
