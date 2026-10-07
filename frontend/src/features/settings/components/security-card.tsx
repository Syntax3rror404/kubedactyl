import { ShieldIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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
import { Switch } from "@/components/ui/switch"

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
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ShieldIcon className="size-4" />
          Security
        </CardTitle>
        <CardDescription>Network isolation, lifetimes of sign-ins and API tokens, and the API docs.</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-6">
          <FieldSet>
            <FieldLegend variant="label">Network isolation</FieldLegend>
            <Field orientation="horizontal">
              <Switch
                id="isolation"
                checked={!values.allowPrivateNetworks}
                onCheckedChange={(isolated) => onChange({ allowPrivateNetworks: !isolated })}
              />
              <div>
                <FieldLabel htmlFor="isolation">Isolate user namespaces (recommended)</FieldLabel>
                <FieldDescription>
                  Servers only reach the internet, the cluster DNS and their owner's other servers. Players are not
                  affected.
                </FieldDescription>
              </div>
            </Field>
          </FieldSet>
          <FieldSeparator />
          <LifetimeFields values={values} onChange={onChange} errors={errors} />
          <FieldSeparator />
          <FieldSet>
            <FieldLegend variant="label">API documentation</FieldLegend>
            <Field orientation="horizontal">
              <Switch
                id="api-docs"
                checked={!values.disableApiDocs}
                onCheckedChange={(enabled) => onChange({ disableApiDocs: !enabled })}
              />
              <FieldLabel htmlFor="api-docs">Serve the API documentation (Swagger UI)</FieldLabel>
            </Field>
          </FieldSet>
        </FieldGroup>
      </CardContent>
    </Card>
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
        <Field data-invalid={errors.sessionHours ? true : undefined}>
          <FieldLabel htmlFor="session-hours">Sign-in lasts (hours)</FieldLabel>
          <Input
            id="session-hours"
            type="number"
            min={1}
            max={720}
            value={values.sessionHours}
            onChange={(e) => onChange({ sessionHours: e.target.valueAsNumber || 0 })}
            aria-invalid={errors.sessionHours ? true : undefined}
          />
          {errors.sessionHours ? (
            <FieldError>{errors.sessionHours}</FieldError>
          ) : (
            <FieldDescription>After that, users sign in again (1-720 hours).</FieldDescription>
          )}
        </Field>
        <Field data-invalid={errors.apiTokenMaxDays ? true : undefined}>
          <FieldLabel htmlFor="token-days">API tokens last at most (days)</FieldLabel>
          <Input
            id="token-days"
            type="number"
            min={1}
            max={3650}
            value={values.apiTokenMaxDays}
            onChange={(e) => onChange({ apiTokenMaxDays: e.target.valueAsNumber || 0 })}
            aria-invalid={errors.apiTokenMaxDays ? true : undefined}
          />
          {errors.apiTokenMaxDays ? (
            <FieldError>{errors.apiTokenMaxDays}</FieldError>
          ) : (
            <FieldDescription>Counted from creation, also for existing tokens (1-3650 days).</FieldDescription>
          )}
        </Field>
      </FieldGroup>
    </FieldSet>
  )
}
