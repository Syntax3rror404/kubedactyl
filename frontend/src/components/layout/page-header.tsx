import type { ReactNode } from "react"

/** Title (with optional badges next to it), description and actions at the top of a page. */
export function PageHeader({
  title,
  badges,
  description,
  actions,
  icon,
}: {
  title: ReactNode
  badges?: ReactNode
  description?: ReactNode
  actions?: ReactNode
  icon?: ReactNode
}) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="flex items-center gap-4">
        {icon}
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="bg-gradient-to-b from-foreground to-foreground/70 bg-clip-text text-2xl font-semibold tracking-tight text-transparent sm:text-3xl">
              {title}
            </h1>
            {badges}
          </div>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  )
}
