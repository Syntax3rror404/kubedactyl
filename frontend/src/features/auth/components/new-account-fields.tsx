import { UsernameField, UsernameHint, usernameRule } from "@/components/common/username-field"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

/**
 * Username and display name of a new account (invite and first setup); error is the server's message for the
 * username, a fixed username was chosen by the administrator who sent the invite.
 */
export function NewAccountFields({
  username,
  displayName,
  onUsername,
  onDisplayName,
  error,
  fixedUsername,
  autoFocus,
  displayPlaceholder = "optional",
}: {
  username: string
  displayName: string
  onUsername: (value: string) => void
  onDisplayName: (value: string) => void
  error?: string
  fixedUsername?: boolean
  // Focuses the username, or the display name when the username is fixed.
  autoFocus?: boolean
  displayPlaceholder?: string
}) {
  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2">
        <UsernameField
          value={username}
          onChange={onUsername}
          error={error}
          hint={null}
          readOnly={fixedUsername}
          autoFocus={autoFocus && !fixedUsername}
        />
        <Field>
          <FieldLabel htmlFor="display">Display name</FieldLabel>
          <Input
            id="display"
            autoFocus={autoFocus && fixedUsername}
            value={displayName}
            onChange={(e) => onDisplayName(e.target.value)}
            placeholder={displayPlaceholder}
          />
        </Field>
      </div>
      <UsernameHint
        error={error}
        hint={fixedUsername ? "Your administrator chose the username." : `${usernameRule} It cannot be changed later.`}
      />
    </>
  )
}
