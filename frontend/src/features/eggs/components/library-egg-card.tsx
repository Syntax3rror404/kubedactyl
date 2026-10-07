import { Badge } from "@/components/ui/badge"
import { EggCardLayout } from "@/features/eggs/components/egg-card"
import { folderOf, libraryEggPath } from "@/features/eggs/lib/library"
import type { LibraryEgg } from "@/lib/types"

/** Card of an egg in the library (laid out like the cards of installed eggs). */
export function LibraryEggCard({ egg, installed }: { egg: LibraryEgg; installed: boolean }) {
  return (
    <EggCardLayout
      to={libraryEggPath(egg)}
      name={egg.name}
      icon={egg.icon}
      author={egg.author}
      description={egg.description}
      tag={
        <Badge variant="outline" className="font-mono text-[10px]">
          {egg.format}
        </Badge>
      }
      highlight={installed ? "Installed" : undefined}
    >
      <span className="min-w-0 flex-1 truncate font-mono text-muted-foreground">{folderOf(egg.path)}</span>
    </EggCardLayout>
  )
}
