import { TimerIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type Lifetimes = { sessionHours: number; apiTokenMaxDays: number }

/** How long sign-ins and API tokens stay valid. */
export function LifetimesCard({
  values,
  onChange,
  errors,
}: {
  values: Lifetimes
  onChange: (patch: Partial<Lifetimes>) => void
  errors: Record<string, string>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TimerIcon className="size-4" />
          Sessions and API tokens
        </CardTitle>
        <CardDescription>
          Shorter lifetimes limit the damage of a stolen session or token. Changes also apply to existing ones.
        </CardDescription>
      </CardHeader>
      <CardContent>
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
      </CardContent>
    </Card>
  )
}
