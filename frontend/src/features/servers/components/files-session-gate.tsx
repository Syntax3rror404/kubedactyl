import { MoonIcon, PlayIcon } from "lucide-react"

import { ProgressStripe } from "@/components/common/callout"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { plural } from "@/lib/format"
import type { useFilesSession } from "@/lib/queries"

/**
 * Shows its children once the file container is ready; until then a placeholder while it starts, or
 * a note after it was stopped for inactivity.
 */
export function FilesSessionGate({
  files,
  children,
}: {
  files: ReturnType<typeof useFilesSession>
  children: React.ReactNode
}) {
  const { view, session, error } = files
  if (view === "ready") return children
  const seconds = session?.idleTimeoutSeconds ?? 60
  const idle = seconds % 60 === 0 ? plural(seconds / 60, "minute") : plural(seconds, "second")
  return (
    <Card className="relative overflow-hidden py-0">
      {view === "starting" && <ProgressStripe />}
      <div className="space-y-2 p-4 opacity-60" aria-hidden>
        {Array.from({ length: 7 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="size-4 rounded" />
            <Skeleton className="h-4" style={{ width: `${30 + ((i * 17) % 45)}%` }} />
            <Skeleton className="ml-auto h-4 w-16" />
          </div>
        ))}
      </div>
      <div className="absolute inset-0 flex items-center justify-center bg-background/40 backdrop-blur-[2px]">
        <div className="mx-4 flex max-w-sm flex-col items-center gap-3 rounded-2xl border bg-card/90 px-6 py-5 text-center shadow-lg">
          {view === "starting" ? (
            <>
              <Spinner className="size-6 text-sky-500" />
              <p className="font-medium">Please wait a few seconds, your file container is spinning up…</p>
              <p className="font-mono text-xs text-muted-foreground">
                {error ? error.message : (session?.message ?? "requesting")}
              </p>
            </>
          ) : (
            <>
              <MoonIcon className="size-6 text-muted-foreground" />
              <p className="font-medium">The file container was stopped</p>
              <p className="text-sm text-muted-foreground">
                It stops after {idle} without file activity so idle servers don't keep pods running.
              </p>
              <Button size="sm" onClick={files.restart}>
                <PlayIcon />
                Start again
              </Button>
            </>
          )}
        </div>
      </div>
    </Card>
  )
}
