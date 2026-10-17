import { ShieldIcon } from "lucide-react"

import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { SettingsCard, SwitchField } from "@/features/settings/components/settings-card"

type Security = {
  allowPrivateNetworks: boolean
  sessionHours: number
  apiTokenMaxDays: number
  disableApiDocs: boolean
}

/**
 * Network isolation of user namespaces (NetworkPolicy kubedactyl-isolation), how long sign-ins and API tokens stay
 * valid, and whether the API documentation (Swagger UI at /swagger/) is served.
 */
export function SecurityCard({
  values,
  onChange,
  errors,
}: {
  values: Security
  onChange: (patch: Partial<Security>) => void
  errors: Record<string, string>
}) {
  return (
    <SettingsCard
      icon={ShieldIcon}
      title="Security"
      description="Network isolation, lifetimes of sign-ins and API tokens, and the API docs."
    >
      <FieldGroup className="gap-6">
        <FieldSet>
          <FieldLegend variant="label">Network isolation</FieldLegend>
          <SwitchField
            id="isolation"
            label="Isolate user namespaces (recommended)"
            checked={!values.allowPrivateNetworks}
            onChange={(isolated) => onChange({ allowPrivateNetworks: !isolated })}
          >
            Servers only reach the internet, the cluster DNS and their owner's other servers. Players are not affected.
          </SwitchField>
        </FieldSet>
        <FieldSeparator />
        <LifetimeFields values={values} onChange={onChange} errors={errors} />
        <FieldSeparator />
        <FieldSet>
          <FieldLegend variant="label">API documentation</FieldLegend>
          <SwitchField
            id="api-docs"
            label="Enable Swagger UI"
            checked={!values.disableApiDocs}
            onChange={(enabled) => onChange({ disableApiDocs: !enabled })}
          />
        </FieldSet>
      </FieldGroup>
    </SettingsCard>
  )
}

/** Sign-in and API token lifetimes; shorter values also end existing sessions and tokens. */
function LifetimeFields({
  values,
  onChange,
  errors,
}: {
  values: Security
  onChange: (patch: Partial<Security>) => void
  errors: Record<string, string>
}) {
  return (
    <FieldSet>
      <FieldLegend variant="label">Sessions and API tokens</FieldLegend>
      <FieldGroup className="grid gap-6 sm:grid-cols-2">
        <LifetimeField
          id="session-hours"
          label="Sign-in lasts (hours)"
          hint="After that, users sign in again (1-720 hours)."
          max={720}
          value={values.sessionHours}
          onChange={(sessionHours) => onChange({ sessionHours })}
          error={errors.sessionHours}
        />
        <LifetimeField
          id="token-days"
          label="API tokens last at most (days)"
          hint="Counted from creation, also for existing tokens (1-3650 days)."
          max={3650}
          value={values.apiTokenMaxDays}
          onChange={(apiTokenMaxDays) => onChange({ apiTokenMaxDays })}
          error={errors.apiTokenMaxDays}
        />
      </FieldGroup>
    </FieldSet>
  )
}

/** A lifetime from 1 to max; an empty field counts as 0, which the panel refuses. */
function LifetimeField({
  id,
  label,
  hint,
  max,
  value,
  onChange,
  error,
}: {
  id: string
  label: string
  hint: string
  max: number
  value: number
  onChange: (value: number) => void
  error?: string
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        type="number"
        min={1}
        max={max}
        value={value}
        onChange={(e) => onChange(e.target.valueAsNumber || 0)}
        aria-invalid={error ? true : undefined}
      />
      {error ? <FieldError>{error}</FieldError> : <FieldDescription>{hint}</FieldDescription>}
    </Field>
  )
}
