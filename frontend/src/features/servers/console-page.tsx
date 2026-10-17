import { useOutletContext } from "react-router"

import { ConsoleWindow } from "@/features/servers/components/console-window"
import { StatsPanel } from "@/features/servers/components/stats-panel"
import { EggFeatures } from "@/features/servers/egg-features/egg-features"
import { useConsole } from "@/features/servers/hooks/use-console"
import type { ServerContext } from "@/features/servers/server-layout"
import { useAuth } from "@/hooks/use-auth"
import { activePhases, phaseOf, serverAccess } from "@/lib/format"

/**
 * /servers/:server: live console, command input, stats and the egg feature dialogs. A subuser who may only view the
 * server sends no commands, and the dialogs (they change files) are for the owner.
 */
export function ConsolePage() {
  const { server, egg } = useOutletContext<ServerContext>()
  const { user } = useAuth()
  const name = server.metadata.name
  const console_ = useConsole(name)
  const access = serverAccess(server, user)
  const active = activePhases.includes(phaseOf(server))

  return (
    <div className="grid gap-6 xl:grid-cols-[1fr_20rem]">
      <ConsoleWindow title={`${name} console`} console={console_} canSend={active && access !== "view"} />
      <StatsPanel server={server} />
      {egg && access === "full" && <EggFeatures server={server} egg={egg} subscribe={console_.subscribe} />}
    </div>
  )
}
