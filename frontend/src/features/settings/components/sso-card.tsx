import { KeyRoundIcon } from "lucide-react"

import { CopyButton } from "@/components/common/copy-button"
import { UnveilPassword } from "@/components/common/unveil-password"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import type { OIDCSettings } from "@/lib/types"

type TextField = {
  key: "name" | "issuerUrl" | "clientId" | "adminGroup" | "userGroup" | "groupsClaim" | "usernameClaim"
  label: string
  placeholder: string
  hint?: string
}

const textFields: TextField[] = [
  { key: "issuerUrl", label: "Issuer URL", placeholder: "https://auth.example.com/realms/games" },
  { key: "clientId", label: "Client ID", placeholder: "kubedactyl" },
  { key: "adminGroup", label: "Admin group", placeholder: "kubedactyl-admins", hint: "Members are administrators." },
  {
    key: "userGroup",
    label: "User group",
    placeholder: "kubedactyl-users",
    hint: "Members are users. Everyone else is refused.",
  },
  { key: "groupsClaim", label: "Groups claim", placeholder: "groups" },
  {
    key: "usernameClaim",
    label: "Username claim",
    placeholder: "preferred_username",
    hint: "Becomes the panel username; it must already be valid (lowercase).",
  },
  { key: "name", label: "Button name", placeholder: "Keycloak", hint: "The button reads: Sign in with <name>." },
]

/**
 * Single sign-on through an OpenID Connect identity provider: client, the groups that grant access, how existing
 * accounts are linked and the redirect URL to register (filled with this address when it is turned on). The client
 * secret is write only (the panel never sends it back).
 */
export function SsoCard({
  values,
  onChange,
  secret,
  secretSet,
  onSecret,
  errors,
}: {
  values: OIDCSettings
  onChange: (patch: Partial<OIDCSettings>) => void
  secret: string
  secretSet: boolean
  onSecret: (secret: string) => void
  errors: Record<string, string>
}) {
  const thisPanel = `${location.origin}/api/auth/oidc/callback`
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <KeyRoundIcon className="size-4" />
          Single sign-on
        </CardTitle>
        <CardDescription>Sign in through an OpenID Connect identity provider. Its groups grant access.</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-6">
          <Field orientation="horizontal">
            <Switch
              id="oidc-enabled"
              checked={!!values.enabled}
              onCheckedChange={(enabled) => onChange({ enabled, redirectUrl: values.redirectUrl || thisPanel })}
            />
            <FieldLabel htmlFor="oidc-enabled">Offer single sign-on on the sign-in page</FieldLabel>
          </Field>
          <FieldGroup className="grid gap-6 md:grid-cols-2">
            {textFields.map((f) => (
              <TextInput
                key={f.key}
                field={f}
                value={values[f.key] ?? ""}
                onChange={(v) => onChange({ [f.key]: v })}
                error={errors[`oidc.${f.key}`]}
              />
            ))}
            <Field data-invalid={errors.oidcClientSecret ? true : undefined}>
              <FieldLabel htmlFor="oidc-secret">Client secret</FieldLabel>
              <UnveilPassword
                id="oidc-secret"
                autoComplete="off"
                value={secret}
                onChange={(e) => onSecret(e.target.value)}
                placeholder={secretSet ? "Stored, type to replace" : "Empty for a public client"}
              />
              {errors.oidcClientSecret && <FieldError>{errors.oidcClientSecret}</FieldError>}
            </Field>
          </FieldGroup>
          <FieldSeparator />
          <FieldSet>
            <Field orientation="horizontal">
              <Switch
                id="oidc-link"
                checked={!!values.linkByUsername}
                onCheckedChange={(linkByUsername) => onChange({ linkByUsername })}
              />
              <div>
                <FieldLabel htmlFor="oidc-link">Link existing accounts by username</FieldLabel>
                <FieldDescription>
                  To take over existing accounts. Turn it off afterwards: whoever picks a name at the identity provider
                  gets that account.
                </FieldDescription>
              </div>
            </Field>
            <Field orientation="horizontal">
              <Switch
                id="oidc-keep-passwords"
                checked={!!values.keepPasswords}
                onCheckedChange={(keepPasswords) => onChange({ keepPasswords })}
              />
              <div>
                <FieldLabel htmlFor="oidc-keep-passwords">Keep passwords of linked accounts</FieldLabel>
                <FieldDescription>
                  They can still sign in with their password; their role then changes only at single sign-on.
                </FieldDescription>
              </div>
            </Field>
          </FieldSet>
          <Field data-invalid={errors["oidc.redirectUrl"] ? true : undefined}>
            <FieldLabel htmlFor="oidc-redirect">Redirect URL</FieldLabel>
            <div className="flex items-center gap-2">
              <Input
                id="oidc-redirect"
                value={values.redirectUrl ?? ""}
                onChange={(e) => onChange({ redirectUrl: e.target.value })}
                placeholder={thisPanel}
                className="font-mono"
                aria-invalid={errors["oidc.redirectUrl"] ? true : undefined}
              />
              <CopyButton value={values.redirectUrl || thisPanel} />
            </div>
            {errors["oidc.redirectUrl"] ? (
              <FieldError>{errors["oidc.redirectUrl"]}</FieldError>
            ) : (
              <FieldDescription>
                The address users open the panel with; register it at the identity provider.
              </FieldDescription>
            )}
          </Field>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

function TextInput({
  field,
  value,
  onChange,
  error,
}: {
  field: TextField
  value: string
  onChange: (value: string) => void
  error?: string
}) {
  const id = `oidc-${field.key}`
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id}>{field.label}</FieldLabel>
      <Input
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={field.placeholder}
        className="font-mono"
        aria-invalid={error ? true : undefined}
      />
      {error ? <FieldError>{error}</FieldError> : field.hint && <FieldDescription>{field.hint}</FieldDescription>}
    </Field>
  )
}
