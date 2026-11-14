import { Fragment, type ComponentType } from "react"
import { ShieldIcon } from "lucide-react"

import { NumberField, SettingsCard, SwitchField } from "@/base/features/settings/components/settings-card"
import type { SettingsCardProps } from "@/base/features/settings/lib/settings-draft"
import { FieldGroup, FieldLegend, FieldSeparator, FieldSet } from "@/components/ui/field"

type Fields = ComponentType<SettingsCardProps>[]

/**
 * How long sign-ins and API tokens stay valid and whether the API documentation (Swagger UI at /swagger/) is
 * served; the app's security settings come before (`before`) and after (`after`) the lifetimes, `description` names
 * all of them.
 */
export function SecurityCard({
  description,
  before,
  after,
  ...props
}: SettingsCardProps & { description: string; before: Fields; after: Fields }) {
  const { draft, onChange } = props
  // Each field is followed by a separator; the API documentation comes last.
  const fields = [...before, LifetimeFields, ...after]
  return (
    <SettingsCard icon={ShieldIcon} title="Security" description={description}>
      <FieldGroup className="gap-6">
        {fields.map((Field, i) => (
          <Fragment key={i}>
            <Field {...props} />
            <FieldSeparator />
          </Fragment>
        ))}
        <FieldSet>
          <FieldLegend variant="label">API documentation</FieldLegend>
          <SwitchField
            id="api-docs"
            label="Enable Swagger UI"
            checked={!draft.disableApiDocs}
            onChange={(enabled) => onChange({ disableApiDocs: !enabled })}
          />
        </FieldSet>
      </FieldGroup>
    </SettingsCard>
  )
}

/** Sign-in and API token lifetimes; shorter values also end existing sessions and tokens. */
function LifetimeFields({ draft, onChange, errors }: SettingsCardProps) {
  return (
    <FieldSet>
      <FieldLegend variant="label">Sessions and API tokens</FieldLegend>
      <FieldGroup className="grid gap-6 sm:grid-cols-2">
        <NumberField
          id="session-hours"
          label="Sign-in lasts (hours)"
          hint="After that, users sign in again (1-720 hours)."
          max={720}
          value={draft.sessionHours}
          onChange={(sessionHours) => onChange({ sessionHours })}
          error={errors.sessionHours}
        />
        <NumberField
          id="token-days"
          label="API tokens last at most (days)"
          hint="Counted from creation, also for existing tokens (1-3650 days)."
          max={3650}
          value={draft.apiTokenMaxDays}
          onChange={(apiTokenMaxDays) => onChange({ apiTokenMaxDays })}
          error={errors.apiTokenMaxDays}
        />
      </FieldGroup>
    </FieldSet>
  )
}
