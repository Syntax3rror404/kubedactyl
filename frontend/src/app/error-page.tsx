import { CircleAlertIcon, RefreshCwIcon } from "lucide-react"
import { useRouteError } from "react-router"

import { Button } from "@/components/ui/button"

/** Shown when a page cannot be loaded or shown; reloading also fetches the pages of a newer build. */
export function ErrorPage() {
  const error = useRouteError()
  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-4 text-center">
      <CircleAlertIcon className="size-8 text-destructive" />
      <div className="space-y-1">
        <p className="font-medium">Could not show the page</p>
        {error instanceof Error && (
          <p className="max-w-md text-sm wrap-break-word text-muted-foreground">{error.message}</p>
        )}
      </div>
      <Button variant="outline" onClick={() => window.location.reload()}>
        <RefreshCwIcon />
        Reload
      </Button>
    </div>
  )
}
