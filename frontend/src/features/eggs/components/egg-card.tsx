import { Link } from "react-router"

import { EggIcon } from "@/components/common/egg-icon"
import { GlowCard } from "@/components/common/glow-card"
import { Badge } from "@/components/ui/badge"
import { EggOrigin } from "@/features/eggs/components/egg-origin"
import { UpdateUrlBadge } from "@/features/eggs/components/update-url-badge"
import { plural } from "@/lib/format"
import type { Egg } from "@/lib/types"

/** Card of an egg on the eggs page. */
export function EggCard({ egg, servers }: { egg: Egg; servers: number }) {
  const s = egg.spec
  return (
    <Link
      to={`/eggs/${egg.metadata.name}`}
      className="block h-full rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <GlowCard>
        <div className="flex h-full flex-col gap-4 p-5">
          <div className="flex items-start gap-3">
            <EggIcon name={s.displayName} icon={s.icon} className="size-12" />
            <div className="min-w-0 flex-1">
              <h3 className="truncate font-semibold tracking-tight">{s.displayName}</h3>
              <p className="truncate text-xs text-muted-foreground">{s.author || "unknown author"}</p>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">
                <EggOrigin spec={s} className="font-mono text-[10px]" />
              </div>
            </div>
          </div>
          <p className="line-clamp-2 text-sm text-muted-foreground">{s.description || "No description."}</p>
          <div className="mt-auto flex flex-wrap gap-1.5 text-xs">
            <Badge variant="secondary">{plural(s.dockerImages.length, "image")}</Badge>
            <Badge variant="secondary">{plural(s.variables?.length ?? 0, "variable")}</Badge>
            <UpdateUrlBadge spec={s} />
            {servers > 0 && (
              <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">
                {plural(servers, "server")}
              </Badge>
            )}
          </div>
        </div>
      </GlowCard>
    </Link>
  )
}
