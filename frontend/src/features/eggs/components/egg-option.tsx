import { CheckIcon } from "lucide-react"

import { EggIcon } from "@/components/common/egg-icon"
import { plural } from "@/lib/format"
import type { Egg } from "@/lib/types"
import { cn } from "@/lib/utils"

/** Selectable egg tile of the create wizard. */
export function EggOption({ egg, selected, onSelect }: { egg: Egg; selected: boolean; onSelect: () => void }) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "relative flex items-center gap-3 rounded-xl border p-3 text-left transition-all hover:bg-accent/50",
        selected && "border-primary bg-accent/40 ring-2 ring-primary/20",
      )}
    >
      <EggIcon name={egg.spec.displayName} icon={egg.spec.icon} />
      <div className="min-w-0">
        <div className="truncate text-sm font-medium">{egg.spec.displayName}</div>
        <div className="truncate text-xs text-muted-foreground">
          {plural(egg.spec.dockerImages.length, "image")} · {egg.spec.source?.format}
        </div>
      </div>
      {selected && <CheckIcon className="absolute top-2 right-2 size-4 text-primary" />}
    </button>
  )
}
