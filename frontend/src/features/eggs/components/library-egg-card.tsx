import { Link } from "react-router"

import { EggIcon } from "@/components/common/egg-icon"
import { GlowCard } from "@/components/common/glow-card"
import { Badge } from "@/components/ui/badge"
import { folderOf, libraryEggPath } from "@/features/eggs/lib/library"
import type { LibraryEgg } from "@/lib/types"

/** Card of an egg in the library (laid out like the cards of installed eggs). */
export function LibraryEggCard({ egg, installed }: { egg: LibraryEgg; installed: boolean }) {
  return (
    <Link
      to={libraryEggPath(egg)}
      className="block h-full rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <GlowCard>
        <div className="flex h-full flex-col gap-4 p-5">
          <div className="flex items-start gap-3">
            <EggIcon name={egg.name} icon={egg.icon} className="size-12" />
            <div className="min-w-0 flex-1">
              <h3 className="truncate font-semibold tracking-tight">{egg.name}</h3>
              <p className="truncate text-xs text-muted-foreground">{egg.author || "unknown author"}</p>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">
                <Badge variant="outline" className="font-mono text-[10px]">
                  {egg.format}
                </Badge>
              </div>
            </div>
          </div>
          <p className="line-clamp-2 text-sm text-muted-foreground">{egg.description || "No description."}</p>
          <div className="mt-auto flex items-center gap-2 text-xs">
            <span className="min-w-0 flex-1 truncate font-mono text-muted-foreground">{folderOf(egg.path)}</span>
            {installed && <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">Installed</Badge>}
          </div>
        </div>
      </GlowCard>
    </Link>
  )
}
