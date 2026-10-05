import { useOutletContext } from "react-router"

import { ConsoleWindow } from "@/features/servers/components/console-window"
import { StatsPanel } from "@/features/servers/components/stats-panel"
import { EggFeatures } from "@/features/servers/egg-features/egg-features"
import { useConsole } from "@/features/servers/hooks/use-console"
import type { ServerContext } from "@/features/servers/server-layout"
import { activePhases, phaseOf } from "@/lib/format"

/** /servers/:server: live console, command input, stats and the egg feature dialogs. */
export function ConsolePage() {
  const { server, egg } = useOutletContext<ServerContext>()
  const name = server.metadata.name
  const console_ = useConsole(name)
  const active = activePhases.includes(phaseOf(server))

  return (
    <div className="grid gap-6 xl:grid-cols-[1fr_20rem]">
      <ConsoleWindow title={`${name} console`} console={console_} canSend={active} />
      <StatsPanel server={server} />
      {/* Egg features (Pterodactyl "features" field) */}
      {egg && <EggFeatures server={server} egg={egg} subscribe={console_.subscribe} />}
    </div>
  )
}
