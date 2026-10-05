import { RefreshCwIcon, StethoscopeIcon } from "lucide-react"
import { useOutletContext } from "react-router"

import { CheckStatusIcon } from "@/components/common/check-status-icon"
import { QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type { ServerContext } from "@/features/servers/server-layout"
import { formatRelativeTime } from "@/lib/format"
import { useServerDiagnostics } from "@/lib/queries"

/** /servers/:server/diagnostics: the typical reasons why players cannot connect, checked one by one. */
export function DiagnosticsPage() {
  const { server } = useOutletContext<ServerContext>()
  const diagnostics = useServerDiagnostics(server.metadata.name)
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="space-y-1.5">
          <CardTitle className="flex items-center gap-2">
            <StethoscopeIcon className="size-4" />
            Diagnostics
          </CardTitle>
          <CardDescription>Common reasons why players cannot connect. UDP ports cannot be tested.</CardDescription>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={diagnostics.isFetching}
          onClick={() => void diagnostics.refetch()}
        >
          <RefreshCwIcon className={diagnostics.isFetching ? "animate-spin" : ""} />
          Run again
        </Button>
      </CardHeader>
      <CardContent>
        <QueryState query={diagnostics} skeleton={<Skeleton className="h-72 rounded-xl" />}>
          {(data) => (
            <div className="space-y-3">
              <ul className="divide-y rounded-xl border">
                {data.checks.map((check) => (
                  <li key={check.id} className="flex gap-3 p-3">
                    <CheckStatusIcon status={check.status} className="mt-0.5 size-4" />
                    <div className="min-w-0">
                      <div className="text-sm font-medium">{check.label}</div>
                      {check.message && <p className="text-sm text-muted-foreground">{check.message}</p>}
                    </div>
                  </li>
                ))}
              </ul>
              <p className="text-xs text-muted-foreground">Checked {formatRelativeTime(data.checkedAt)}</p>
            </div>
          )}
        </QueryState>
      </CardContent>
    </Card>
  )
}
