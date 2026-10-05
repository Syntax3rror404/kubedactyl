import { DetailList } from "@/components/common/detail-list"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { FieldError } from "@/components/ui/field"
import { ResourceSliders } from "@/features/servers/components/resource-sliders"
import type { SettingsChange, SettingsDraft } from "@/features/servers/lib/settings-draft"
import { formatCpuLimit, formatMiB, serverAddress } from "@/lib/format"
import type { GameServer } from "@/lib/types"

/** Settings page: memory, CPU and disk sliders (administrators). */
export function ResourcesCard({
  server,
  draft,
  onChange,
  errors,
}: {
  server: GameServer
  draft: SettingsDraft
  onChange: SettingsChange
  errors: Record<string, string>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Resources</CardTitle>
        <CardDescription>
          Disk can only grow (online volume expansion). Storage class{" "}
          <span className="font-mono">{server.spec.storageClass ?? "default"}</span>.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ResourceSliders
          memory={draft.memory}
          onMemoryChange={(memory) => onChange({ memory })}
          cpu={draft.cpu}
          onCpuChange={(cpu) => onChange({ cpu })}
          disk={draft.disk}
          onDiskChange={(disk) => onChange({ disk })}
          minDisk={server.spec.resources.diskMiB}
        />
        {["memoryMiB", "cpuMillis", "diskMiB"].map(
          (k) =>
            errors[k] && (
              <FieldError key={k} className="mt-3">
                {errors[k]}
              </FieldError>
            ),
        )}
      </CardContent>
    </Card>
  )
}

/** Settings page: the resources read-only, for users (only administrators change them). */
export function ResourceSummary({ server, domain }: { server: GameServer; domain?: string }) {
  const r = server.spec.resources
  return (
    <Card>
      <CardHeader>
        <CardTitle>Resources</CardTitle>
        <CardDescription>Assigned by your administrator.</CardDescription>
      </CardHeader>
      <CardContent>
        <DetailList
          rows={[
            ["Memory", formatMiB(r.memoryMiB)],
            ["CPU", formatCpuLimit(r.cpuMillis)],
            ["Disk", formatMiB(r.diskMiB)],
            ["Storage class", server.spec.storageClass ?? "default"],
            ["Ports", server.spec.ports.join(", ")],
            ["Address", serverAddress(server, domain) ?? "pending"],
          ]}
        />
      </CardContent>
    </Card>
  )
}
