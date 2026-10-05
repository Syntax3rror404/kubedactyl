import type { ReactNode } from "react"
import type { UseQueryResult } from "@tanstack/react-query"
import { CircleAlertIcon, RefreshCwIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

/**
 * The one way to show data of a query: `skeleton` while it loads, the error with a retry button
 * when the request failed, `empty` when there is nothing (by default: an empty list), otherwise
 * children(data). Pages render their PageHeader outside, so it stays visible in every state.
 */
export function QueryState<T>({
  query,
  skeleton = <Skeleton className="h-64 rounded-2xl" />,
  empty,
  isEmpty = (data) => Array.isArray(data) && data.length === 0,
  children,
}: {
  query: Pick<UseQueryResult<T>, "data" | "error" | "refetch">
  skeleton?: ReactNode
  empty?: ReactNode
  isEmpty?: (data: T) => boolean
  children: (data: T) => ReactNode
}) {
  if (query.data === undefined) {
    return query.error ? <ErrorState error={query.error} onRetry={() => void query.refetch()} /> : skeleton
  }
  if (empty !== undefined && isEmpty(query.data)) return empty
  return children(query.data)
}

/** A request that failed, with its message and a retry button. */
function ErrorState({ error, onRetry }: { error: Error; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 p-8 text-center">
      <CircleAlertIcon className="size-6 text-destructive" />
      <div className="space-y-1">
        <p className="font-medium">Could not load the data</p>
        <p className="text-sm text-muted-foreground">{error.message}</p>
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCwIcon />
          Try again
        </Button>
      )}
    </div>
  )
}

/**
 * Nothing to show yet. "page" is a dashed box on the page, "card" sits inside a card or table.
 * children are the actions (buttons).
 */
export function EmptyState({
  icon,
  title,
  description,
  variant = "page",
  children,
}: {
  icon?: ReactNode
  title: string
  description?: ReactNode
  variant?: "page" | "card"
  children?: ReactNode
}) {
  return (
    <Empty className={cn(variant === "page" ? "rounded-2xl border border-dashed bg-card/40 py-16" : "py-14")}>
      <EmptyHeader>
        {icon && <EmptyMedia variant="icon">{icon}</EmptyMedia>}
        <EmptyTitle>{title}</EmptyTitle>
        {description && <EmptyDescription>{description}</EmptyDescription>}
      </EmptyHeader>
      {children && <EmptyContent className="flex-row justify-center">{children}</EmptyContent>}
    </Empty>
  )
}

/** Placeholder cards in the grid layout of the list pages. */
export function SkeletonGrid({ className = "h-44" }: { className?: string }) {
  return (
    <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} className={cn("rounded-2xl", className)} />
      ))}
    </div>
  )
}
