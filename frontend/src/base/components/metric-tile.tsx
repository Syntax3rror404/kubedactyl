import { BRAND_FROM, BRAND_TO } from "@/app/config"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

/** Big number with a label for the header rows of overview pages; accent paints it in the brand gradient. */
export function MetricTile({
  icon,
  label,
  value,
  hint,
  accent,
}: {
  icon: React.ReactNode
  label: string
  value: string | null
  hint?: string
  accent?: boolean
}) {
  return (
    <div className="relative overflow-hidden rounded-xl border bg-card/60 p-4 backdrop-blur-sm">
      <div className="flex items-center justify-between gap-2 text-muted-foreground [&_svg]:size-4 [&_svg]:shrink-0">
        {/* One line: a wrapped label would put the value lower than in the tiles next to it. */}
        <span className="truncate text-xs font-medium tracking-wide uppercase" title={label}>
          {label}
        </span>
        {icon}
      </div>
      <div className="mt-2 flex flex-wrap items-baseline gap-x-2">
        {value == null ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <span
            className={cn(
              "text-2xl font-semibold whitespace-nowrap sm:text-3xl",
              accent && "bg-clip-text text-transparent",
            )}
            style={accent ? { backgroundImage: `linear-gradient(to right, ${BRAND_FROM}, ${BRAND_TO})` } : undefined}
          >
            {value}
          </span>
        )}
        {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      </div>
    </div>
  )
}
