import { NumberField, SwitchField } from "@/base/features/settings/settings-card"
import type { SettingsCardProps } from "@/base/features/settings/settings-draft"
import { FieldLegend, FieldSet } from "@/components/ui/field"

/** Network isolation of user namespaces (NetworkPolicy kubedactyl-isolation). */
export function IsolationField({ draft, onChange }: SettingsCardProps) {
  return (
    <FieldSet>
      <FieldLegend variant="label">Network isolation</FieldLegend>
      <SwitchField
        id="isolation"
        label="Isolate user namespaces (recommended)"
        checked={!draft.allowPrivateNetworks}
        onChange={(isolated) => onChange({ allowPrivateNetworks: !isolated })}
      >
        Servers only reach the internet, the cluster DNS and their owner's other servers. Players are not affected.
      </SwitchField>
    </FieldSet>
  )
}

/** How many subusers every user may invite. */
export function SubusersField({ draft, onChange, errors }: SettingsCardProps) {
  return (
    <FieldSet>
      <FieldLegend variant="label">Subusers</FieldLegend>
      <NumberField
        id="subusers"
        label="Subusers per user"
        hint="Accounts a user may invite to watch their servers (0-100, 0 = off)."
        min={0}
        max={100}
        value={draft.subusersPerUser}
        onChange={(subusersPerUser) => onChange({ subusersPerUser })}
        error={errors.subusersPerUser}
      />
    </FieldSet>
  )
}
