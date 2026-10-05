import { Skeleton } from "@/components/ui/skeleton"

/** Big number with a label, used in the header rows of the dashboard and the cluster page. */
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
      <div className="flex items-center justify-between text-muted-foreground [&_svg]:size-4">
        <span className="text-xs font-medium tracking-wide uppercase">{label}</span>
        {icon}
      </div>
      <div className="mt-2 flex flex-wrap items-baseline gap-x-2">
        {value == null ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <span
            className={
              accent
                ? "bg-gradient-to-r from-emerald-500 to-sky-500 bg-clip-text text-2xl font-semibold whitespace-nowrap text-transparent sm:text-3xl"
                : "text-2xl font-semibold whitespace-nowrap sm:text-3xl"
            }
          >
            {value}
          </span>
        )}
        {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      </div>
    </div>
  )
}
