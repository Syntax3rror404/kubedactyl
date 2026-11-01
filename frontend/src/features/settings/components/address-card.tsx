import { ArrowRightIcon, GlobeIcon } from "lucide-react"

import { Input } from "@/components/ui/input"
import { SettingsCard, SettingsField } from "@/features/settings/components/settings-card"
import type { SettingsCardProps } from "@/features/settings/lib/settings-draft"
import { useServers } from "@/lib/queries"

/** External domain that users see instead of the load balancer IP. */
export function AddressCard({ draft, onChange, errors }: SettingsCardProps) {
  const servers = useServers()
  const exampleIP = servers.data?.find((s) => s.status?.address && s.spec.serviceType !== "ClusterIP")?.status?.address
  const ip = exampleIP ?? "192.168.1.61"
  const shown = draft.externalDomain.trim().toLowerCase().replace(/\.$/, "")
  return (
    <SettingsCard
      icon={GlobeIcon}
      title="Server address"
      description="Users see this domain instead of the load balancer IP. Administrators still see the IP."
      contentClassName="space-y-4"
    >
      <SettingsField
        id="domain"
        label="External domain"
        hint="Must point to the IPs of the enabled pools. Empty shows the IP."
        error={errors.externalDomain}
      >
        {(control) => (
          <Input
            {...control}
            value={draft.externalDomain}
            onChange={(e) => onChange({ externalDomain: e.target.value })}
            placeholder="play.example.com"
            className="font-mono"
          />
        )}
      </SettingsField>
      <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-muted/40 px-3 py-2 font-mono text-sm">
        <span className={shown ? "text-muted-foreground line-through" : undefined}>{ip}:25565</span>
        {shown && (
          <>
            <ArrowRightIcon className="size-4 text-muted-foreground" />
            <span className="font-medium">{shown}:25565</span>
          </>
        )}
      </div>
    </SettingsCard>
  )
}
