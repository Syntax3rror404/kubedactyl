import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { EggSpec } from "@/lib/types"

/** The config file replacements an egg applies before every start. */
export function EggConfigFiles({ spec: s }: { spec: EggSpec }) {
  return (
    <>
      {s.configFiles?.length ? (
        s.configFiles.map((f) => (
          <Card key={f.file}>
            <CardHeader>
              <CardTitle className="font-mono text-sm">{f.file}</CardTitle>
              <CardDescription>{f.parser} parser · applied before every start</CardDescription>
            </CardHeader>
            <CardContent className="space-y-1.5">
              {f.replace?.map((r, i) => (
                <div key={i} className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 font-mono text-xs">
                  <span className="truncate">
                    {r.match}
                    {r.ifValue && <span className="text-muted-foreground"> = {r.ifValue}</span>}
                  </span>
                  <span className="text-muted-foreground">→</span>
                  <span className="truncate text-emerald-600 dark:text-emerald-400">{r.replaceWith || '""'}</span>
                </div>
              ))}
            </CardContent>
          </Card>
        ))
      ) : (
        <p className="text-sm text-muted-foreground">This egg does not modify any config files.</p>
      )}
    </>
  )
}
