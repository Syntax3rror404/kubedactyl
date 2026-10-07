import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

/** The rule for usernames (checked by the server: tenancy.go). */
export const usernameRule = "3-32 lowercase letters, digits or dashes, starting with a letter."

/**
 * The username of a new account: typed in lowercase (the only case usernames have), with the server's error or a
 * hint below. hint={null} leaves both out, for a field in a row whose UsernameHint goes below the whole row.
 *
 * Usage:
 *   <UsernameField value={draft.username} onChange={(v) => set("username", v)} error={errors.username} />
 *   <UsernameField id="i-name" value={…} onChange={…} error={…} hint={null} />  … row …  <UsernameHint error={…} />
 */
export function UsernameField({
  id = "username",
  value,
  onChange,
  error,
  hint = usernameRule,
  readOnly,
  autoFocus,
  placeholder,
}: {
  id?: string
  value: string
  onChange: (value: string) => void
  error?: string
  hint?: string | null
  readOnly?: boolean
  autoFocus?: boolean
  placeholder?: string
}) {
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>Username</FieldLabel>
      <Input
        id={id}
        autoComplete="username"
        autoFocus={autoFocus}
        readOnly={readOnly}
        value={value}
        onChange={(e) => onChange(e.target.value.toLowerCase())}
        placeholder={placeholder}
        className="font-mono"
        aria-invalid={!!error}
      />
      {hint !== null && <UsernameHint error={error} hint={hint} />}
    </Field>
  )
}

/** The server's error for the username, or a hint (by default the rule for usernames). */
export function UsernameHint({ error, hint = usernameRule }: { error?: string; hint?: string }) {
  return error ? <FieldError>{error}</FieldError> : <FieldDescription>{hint}</FieldDescription>
}
