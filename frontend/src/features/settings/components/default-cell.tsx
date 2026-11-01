import { StarIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { Selection } from "@/features/settings/lib/settings-draft"

/** Shows the default badge, or a button that makes an enabled entry the default. */
export function DefaultCell({
  name,
  selection,
  onDefault,
}: {
  name: string
  selection: Selection
  onDefault: (name: string) => void
}) {
  if (selection.defaultName === name) {
    return (
      <Badge className="gap-1">
        <StarIcon className="fill-current" />
        Default
      </Badge>
    )
  }
  if (!selection.enabled.includes(name)) return null
  return (
    <Button variant="ghost" size="sm" className="h-7 text-xs text-muted-foreground" onClick={() => onDefault(name)}>
      Make default
    </Button>
  )
}
