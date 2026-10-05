import { CopyButton } from "@/components/common/copy-button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { EggSpec } from "@/lib/types"

function describeStop(stop?: string) {
  if (!stop) return "Graceful container stop"
  if (stop === "^C" || stop === "^SIGINT") return "Signal SIGINT (^C)"
  if (stop.startsWith("^")) return stop === "^SIGTERM" ? "Signal SIGTERM" : `Signal ${stop} → SIGKILL`
  return `Console command “${stop}”`
}

/** Startup behavior and docker images of an egg. */
export function EggOverview({ spec: s }: { spec: EggSpec }) {
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Startup</CardTitle>
          <CardDescription>Variables in {"{{…}}"} are substituted by the image entrypoint.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-start gap-2 rounded-lg border bg-muted/40 p-3 font-mono text-xs break-all">
            <span className="flex-1">{s.startup}</span>
            <CopyButton value={s.startup} />
          </div>
          <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
            <dt className="text-muted-foreground">Stop</dt>
            <dd>{describeStop(s.stop)}</dd>
            <dt className="text-muted-foreground">Running when</dt>
            <dd className="space-y-1">
              {s.startupDone?.length
                ? s.startupDone.map((d) => (
                    <code key={d} className="block rounded bg-muted px-1.5 py-0.5 text-xs">
                      {d}
                    </code>
                  ))
                : "the container runs"}
            </dd>
            {s.fileDenylist?.length ? (
              <>
                <dt className="text-muted-foreground">Hidden files</dt>
                <dd className="font-mono text-xs">{s.fileDenylist.join(", ")}</dd>
              </>
            ) : null}
          </dl>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Docker images</CardTitle>
          <CardDescription>The first image is the default for new servers.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {s.dockerImages.map((img, i) => (
            <div key={img.image} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
              <div className="min-w-0">
                <div className="text-sm font-medium">
                  {img.name}
                  {i === 0 && (
                    <Badge variant="secondary" className="ml-2">
                      default
                    </Badge>
                  )}
                </div>
                <div className="truncate font-mono text-xs text-muted-foreground">{img.image}</div>
              </div>
              <CopyButton value={img.image} />
            </div>
          ))}
        </CardContent>
      </Card>
    </>
  )
}
