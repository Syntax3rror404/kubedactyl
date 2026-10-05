import { UnveilPassword } from "@/components/common/unveil-password"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { MIN_PASSWORD_LENGTH } from "@/lib/validation"

/** A new password and its confirmation; error is the server's message for the password. */
export function NewPasswordFields({
  password,
  confirm,
  onPassword,
  onConfirm,
  error,
  autoFocus,
  label = "Password",
}: {
  password: string
  confirm: string
  onPassword: (value: string) => void
  onConfirm: (value: string) => void
  error?: string
  autoFocus?: boolean
  label?: string
}) {
  const tooShort = password !== "" && password.length < MIN_PASSWORD_LENGTH
  const mismatch = confirm !== "" && confirm !== password
  return (
    <>
      <Field data-invalid={tooShort || !!error}>
        <FieldLabel htmlFor="password">{label}</FieldLabel>
        <UnveilPassword
          id="password"
          autoComplete="new-password"
          autoFocus={autoFocus}
          value={password}
          onChange={(e) => onPassword(e.target.value)}
          aria-invalid={tooShort || !!error}
        />
        {error ? (
          <FieldError>{error}</FieldError>
        ) : (
          <FieldDescription>At least {MIN_PASSWORD_LENGTH} characters. Stored as Argon2id hash only.</FieldDescription>
        )}
      </Field>
      <Field data-invalid={mismatch}>
        <FieldLabel htmlFor="confirm">Confirm password</FieldLabel>
        <UnveilPassword
          id="confirm"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => onConfirm(e.target.value)}
          aria-invalid={mismatch}
        />
        {mismatch && <FieldError>The passwords do not match.</FieldError>}
      </Field>
    </>
  )
}
