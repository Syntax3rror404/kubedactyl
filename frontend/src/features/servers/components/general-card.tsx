import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import type { SettingsChange, SettingsDraft } from "@/features/servers/lib/settings-draft"

/** Settings page: display name, crash restarts and (admins) the stop timeout. */
export function GeneralCard({
  draft,
  onChange,
  errors,
  isAdmin,
}: {
  draft: SettingsDraft
  onChange: SettingsChange
  errors: Record<string, string>
  isAdmin: boolean
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>General</CardTitle>
      </CardHeader>
      <CardContent>
        <FieldGroup>
          <Field data-invalid={!!errors.displayName}>
            <FieldLabel htmlFor="display">Display name</FieldLabel>
            <Input id="display" value={draft.displayName} onChange={(e) => onChange({ displayName: e.target.value })} />
            {errors.displayName && <FieldError>{errors.displayName}</FieldError>}
          </Field>
          <Field orientation="horizontal">
            <Switch
              id="crash"
              checked={draft.crashRestart}
              onCheckedChange={(crashRestart) => onChange({ crashRestart })}
            />
            <div>
              <FieldLabel htmlFor="crash">Restart after a crash</FieldLabel>
              <FieldDescription>Not twice within 60 seconds.</FieldDescription>
            </div>
          </Field>
          {isAdmin && (
            <Field data-invalid={!!errors.stopTimeoutSeconds}>
              <FieldLabel htmlFor="timeout">Stop timeout (seconds)</FieldLabel>
              <Input
                id="timeout"
                type="number"
                min={1}
                value={draft.stopTimeout}
                onChange={(e) => onChange({ stopTimeout: Number(e.target.value) })}
                className="w-40"
              />
              <FieldDescription>
                After the stop command the process is killed when it is still running.
              </FieldDescription>
              {errors.stopTimeoutSeconds && <FieldError>{errors.stopTimeoutSeconds}</FieldError>}
            </Field>
          )}
        </FieldGroup>
      </CardContent>
    </Card>
  )
}
