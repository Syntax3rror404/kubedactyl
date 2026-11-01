import { Badge } from "@/components/ui/badge"
import { formatDate } from "@/lib/format"
import type { EggSpec } from "@/lib/types"

/**
 * Where an egg comes from: the export date and format of its file, or "Local" for an egg made in the panel.
 * An imported egg without export date shows only its format.
 */
export function EggOrigin({ spec, className }: { spec: EggSpec; className?: string }) {
  const { exportedAt, format } = spec.source ?? {}
  const label = exportedAt ? `Exported ${formatDate(exportedAt)}` : !format && "Local"
  return (
    <>
      {label && <span className="text-xs whitespace-nowrap text-muted-foreground">{label}</span>}
      {format && (
        <Badge variant="outline" className={className}>
          {format}
        </Badge>
      )}
    </>
  )
}
