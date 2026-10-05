import { PackageIcon } from "lucide-react"

import { DetailList } from "@/components/common/detail-list"
import { QueryState } from "@/components/common/query-state"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useVersions } from "@/lib/queries"

/** Versions of the panel and its main building blocks: Go and Go modules (from the server), frontend libraries (from the build). */
export function VersionsCard() {
  const versions = useVersions()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <PackageIcon className="size-4" />
          Software versions
        </CardTitle>
        <CardDescription>What this panel is built with, useful for bug reports and security notices.</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-6 sm:grid-cols-2">
        <section className="space-y-3">
          <h3 className="text-sm font-medium">Backend</h3>
          <QueryState query={versions} skeleton={<Skeleton className="h-28 rounded-xl" />}>
            {(v) => (
              <DetailList
                rows={[
                  ["Kubedactyl", v.panel],
                  ["Go", v.go.replace(/^go/, "")],
                  ...v.modules.map((m): [string, string] => [m.name, m.version.replace(/^v/, "")]),
                ]}
              />
            )}
          </QueryState>
        </section>
        <section className="space-y-3">
          <h3 className="text-sm font-medium">Frontend</h3>
          <DetailList rows={Object.entries(__FRONTEND_VERSIONS__)} />
        </section>
      </CardContent>
    </Card>
  )
}
