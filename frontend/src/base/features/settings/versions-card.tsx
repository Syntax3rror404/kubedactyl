import { PackageIcon } from "lucide-react"

import { DetailList } from "@/base/components/detail-list"
import { QueryState } from "@/base/components/query-state"
import { SettingsCard } from "@/base/features/settings/settings-card"
import { useVersions } from "@/base/lib/queries"
import { Skeleton } from "@/components/ui/skeleton"

/** Versions of Go and the main modules (from the server) and the frontend libraries (from the build). */
export function VersionsCard() {
  const versions = useVersions()
  return (
    <SettingsCard
      icon={PackageIcon}
      title="Software versions"
      description="What this panel is built with, useful for bug reports and security notices."
      contentClassName="grid gap-6 sm:grid-cols-2"
    >
      <section className="space-y-3">
        <h3 className="text-sm font-medium">Backend</h3>
        <QueryState query={versions} skeleton={<Skeleton className="h-28 rounded-xl" />}>
          {(v) => (
            <DetailList
              rows={[
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
    </SettingsCard>
  )
}
