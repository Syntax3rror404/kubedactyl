import { PasswordField } from "@/components/common/unveil-password"
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
      <PasswordField
        id="password"
        label={label}
        autoComplete="new-password"
        autoFocus={autoFocus}
        value={password}
        onChange={onPassword}
        invalid={tooShort}
        error={error}
        hint={`At least ${MIN_PASSWORD_LENGTH} characters. Stored as Argon2id hash only.`}
      />
      <PasswordField
        id="confirm"
        label="Confirm password"
        autoComplete="new-password"
        value={confirm}
        onChange={onConfirm}
        error={mismatch ? "The passwords do not match." : undefined}
      />
    </>
  )
}
