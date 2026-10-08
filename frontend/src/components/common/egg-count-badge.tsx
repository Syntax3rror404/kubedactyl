import { EggIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { plural } from "@/lib/format"

/** The number of eggs in a repository: settings (egg library) and the Library tab of the eggs page. */
export function EggCountBadge({ count }: { count: number }) {
  return (
    <Badge variant="secondary" className="font-mono" title={plural(count, "egg")}>
      <EggIcon aria-hidden />
      {count}
    </Badge>
  )
}
