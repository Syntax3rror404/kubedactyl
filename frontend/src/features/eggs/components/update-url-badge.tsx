import { RefreshCwIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import type { EggSpec } from "@/lib/types"

/**
 * Shown when the egg has an update URL: "Auto update" when the panel applies newer files by itself,
 * otherwise "Update URL" ("Update from URL" reloads it by hand). The URL is the tooltip.
 */
export function UpdateUrlBadge({ spec }: { spec: EggSpec }) {
  const url = spec.source?.updateUrl
  if (!url) return null
  const auto = !!spec.source?.autoUpdate
  return (
    <Badge
      variant="outline"
      className={
        auto
          ? "gap-1 border-emerald-500/40 text-emerald-700 dark:text-emerald-300"
          : "gap-1 border-sky-500/40 text-sky-700 dark:text-sky-300"
      }
      title={auto ? `Checked every hour: ${url}` : url}
    >
      <RefreshCwIcon className="size-3" />
      {auto ? "Auto update" : "Update URL"}
    </Badge>
  )
}
