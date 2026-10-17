import { PackageIcon } from "lucide-react"

import { DetailList } from "@/components/common/detail-list"
import { QueryState } from "@/components/common/query-state"
import { Skeleton } from "@/components/ui/skeleton"
import { SettingsCard } from "@/features/settings/components/settings-card"
import { useVersions } from "@/lib/queries"

/** Versions of the panel, Go and its main modules (from the server) and the frontend libraries (from the build). */
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
    </SettingsCard>
  )
}
