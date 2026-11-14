import { EggIcon } from "lucide-react"

import { plural } from "@/base/lib/format"
import { Badge } from "@/components/ui/badge"

/** The number of eggs in a repository: settings (egg library) and the Library tab of the eggs page. */
export function EggCountBadge({ count }: { count: number }) {
  return (
    <Badge variant="secondary" className="font-mono" title={plural(count, "egg")}>
      <EggIcon aria-hidden />
      {count}
    </Badge>
  )
}
