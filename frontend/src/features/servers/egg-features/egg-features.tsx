import { Fragment } from "react"

import { PidLimitNotice, SteamDiskSpaceNotice } from "@/features/servers/egg-features/console-notices"
import { EulaPrompt } from "@/features/servers/egg-features/eula-prompt"
import { GslTokenPrompt } from "@/features/servers/egg-features/gsl-token-prompt"
import { HytaleLoginPrompt } from "@/features/servers/egg-features/hytale-login-prompt"
import { JavaVersionPrompt } from "@/features/servers/egg-features/java-version-prompt"
import type { Subscribe } from "@/features/servers/egg-features/use-console-match"
import { supportedFeatures, type EggFeature } from "@/features/servers/lib/egg-features"
import { useAuth } from "@/hooks/use-auth"
import { activePhases, phaseOf } from "@/lib/format"
import type { Egg, GameServer } from "@/lib/types"

interface FeatureProps {
  server: GameServer
  egg: Egg
  subscribe: Subscribe
  active: boolean
  running: boolean
  isAdmin: boolean
}

// One dialog or notice per supported feature (the list in lib/egg-features.ts).
const components: Record<EggFeature, (p: FeatureProps) => React.ReactNode> = {
  eula: (p) => <EulaPrompt server={p.server} subscribe={p.subscribe} active={p.active} />,
  java_version: (p) => <JavaVersionPrompt {...p} />,
  pid_limit: (p) => <PidLimitNotice {...p} />,
  steam_disk_space: (p) => <SteamDiskSpaceNotice {...p} />,
  gsl_token: (p) => <GslTokenPrompt {...p} />,
  hytale_oauth: (p) => <HytaleLoginPrompt subscribe={p.subscribe} running={p.running} />,
}

/**
 * Dialogs for the egg features. Like Pterodactyl they react to console output while the server
 * is not running (a crashed start, a failed update); the history of an earlier run can show them again.
 */
export function EggFeatures({ server, egg, subscribe }: { server: GameServer; egg: Egg; subscribe: Subscribe }) {
  const { isAdmin } = useAuth()
  const features = new Set((egg.spec.features ?? []).map((f) => f.toLowerCase()))
  const phase = phaseOf(server)
  const props = { server, egg, subscribe, active: activePhases.includes(phase), running: phase === "Running", isAdmin }
  return supportedFeatures
    .filter((f) => features.has(f))
    .map((f) => <Fragment key={f}>{components[f](props)}</Fragment>)
}
