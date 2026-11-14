import { KeyRoundIcon, Trash2Icon, Undo2Icon } from "lucide-react"

import { appId, memberGroup, roles } from "@/app/config"
import { CopyButton } from "@/base/components/copy-button"
import { UnveilPassword } from "@/base/components/unveil-password"
import { SettingsCard, SettingsField, SwitchField } from "@/base/features/settings/components/settings-card"
import type { SettingsCardProps, SettingsDraft } from "@/base/features/settings/lib/settings-draft"
import { Button } from "@/components/ui/button"
import { FieldGroup, FieldSeparator, FieldSet } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type TextField = {
  key: "name" | "issuerUrl" | "clientId" | "adminGroup" | typeof memberGroup | "groupsClaim" | "usernameClaim"
  label: string
  placeholder: string
  hint?: string
}

// The default role (app/config.ts) is the one the member group grants.
const member = roles[0]

const textFields: TextField[] = [
  { key: "issuerUrl", label: "Issuer URL", placeholder: `https://auth.example.com/realms/${appId}` },
  { key: "clientId", label: "Client ID", placeholder: appId },
  { key: "adminGroup", label: "Admin group", placeholder: `${appId}-admins`, hint: "Members are administrators." },
  {
    key: memberGroup,
    label: `${member.label} group`,
    placeholder: `${appId}-${member.value}s`,
    hint: `Members are ${member.label.toLowerCase()}s. Everyone else is refused.`,
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

type OIDCDraft = SettingsDraft["oidc"]

/**
 * Single sign-on through an OpenID Connect identity provider: client, the groups that grant access, how existing
 * accounts are linked and the redirect URL to register (filled with this address when it is turned on). The client
 * secret is write only (the panel never sends it back); `secretSet` tells whether one is stored.
 */
export function SsoCard({ draft, onChange, errors, secretSet }: SettingsCardProps & { secretSet: boolean }) {
  const thisPanel = `${location.origin}/api/auth/oidc/callback`
  const values = draft.oidc
  const onOidcChange = (patch: Partial<OIDCDraft>) => onChange({ oidc: { ...values, ...patch } })
  return (
    <SettingsCard
      icon={KeyRoundIcon}
      title="OpenID Connect"
      description="Users sign in through your identity provider."
    >
      <FieldGroup className="gap-6">
        <SwitchField
          id="oidc-enabled"
          label="Enable OIDC on login page"
          checked={!!values.enabled}
          onChange={(enabled) => onOidcChange({ enabled, redirectUrl: values.redirectUrl || thisPanel })}
        />
        <FieldGroup className="grid gap-6 md:grid-cols-2">
          {textFields.map((f) => (
            <TextInput
              key={f.key}
              field={f}
              value={values[f.key]}
              onChange={(v) => onOidcChange({ [f.key]: v })}
              error={errors[`oidc.${f.key}`]}
            />
          ))}
          <SecretField
            secret={draft.oidcClientSecret}
            secretSet={secretSet}
            onSecretChange={(oidcClientSecret) => onChange({ oidcClientSecret })}
            error={errors.oidcClientSecret}
          />
        </FieldGroup>
        <FieldSeparator />
        <FieldSet>
          <SwitchField
            id="oidc-link"
            label="Link existing accounts by username"
            checked={!!values.linkByUsername}
            onChange={(linkByUsername) => onOidcChange({ linkByUsername })}
          >
            To take over existing accounts. Turn it off afterwards: whoever picks a name at the identity provider gets
            that account.
          </SwitchField>
          <SwitchField
            id="oidc-keep-passwords"
            label="Keep passwords of linked accounts"
            checked={!!values.keepPasswords}
            onChange={(keepPasswords) => onOidcChange({ keepPasswords })}
          >
            They can still sign in with their password; their role then changes only at single sign-on.
          </SwitchField>
        </FieldSet>
        <SettingsField id="oidc-redirect" label="Redirect URL" error={errors["oidc.redirectUrl"]}>
          {(control) => (
            <div className="flex items-center gap-2">
              <Input
                {...control}
                value={values.redirectUrl}
                onChange={(e) => onOidcChange({ redirectUrl: e.target.value })}
                placeholder="https://panel.example.com/api/auth/oidc/callback"
                className="font-mono"
              />
              <CopyButton value={values.redirectUrl || thisPanel} />
            </div>
          )}
        </SettingsField>
      </FieldGroup>
    </SettingsCard>
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
  return (
    <SettingsField id={`oidc-${field.key}`} label={field.label} hint={field.hint} error={error}>
      {(control) => (
        <Input
          {...control}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          className="font-mono"
        />
      )}
    </SettingsField>
  )
}

/**
 * The client secret: typing replaces the stored one, an empty field keeps it, Remove deletes it on save
 * (secret "" in the draft, undefined keeps).
 */
function SecretField({
  secret,
  secretSet,
  onSecretChange,
  error,
}: {
  secret: string | undefined
  secretSet: boolean
  onSecretChange: (secret: string | undefined) => void
  error?: string
}) {
  const removing = secret === ""
  return (
    <SettingsField id="oidc-secret" label="Client secret" error={error}>
      {(control) => (
        <div className="flex items-center gap-2">
          <UnveilPassword
            {...control}
            autoComplete="off"
            value={secret ?? ""}
            onChange={(e) => onSecretChange(e.target.value || undefined)}
            placeholder={
              removing ? "Removed when you save" : secretSet ? "Stored, type to replace" : "Empty for a public client"
            }
          />
          {secretSet && !secret && (
            <Button type="button" variant="ghost" size="sm" onClick={() => onSecretChange(removing ? undefined : "")}>
              {removing ? <Undo2Icon /> : <Trash2Icon />}
              {removing ? "Keep" : "Remove"}
            </Button>
          )}
        </div>
      )}
    </SettingsField>
  )
}
