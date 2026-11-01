import { PasswordField } from "@/components/common/unveil-password"
import { MIN_PASSWORD_LENGTH, PASSWORD_RULE } from "@/lib/validation"

/** A new password and its confirmation; error is the server's message for the password. */
export function NewPasswordFields({
  password,
  confirm,
  onPasswordChange,
  onConfirmChange,
  error,
  autoFocus,
  label = "Password",
}: {
  password: string
  confirm: string
  onPasswordChange: (value: string) => void
  onConfirmChange: (value: string) => void
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
        onChange={onPasswordChange}
        invalid={tooShort}
        error={error}
        hint={PASSWORD_RULE}
      />
      <PasswordField
        id="confirm"
        label="Confirm password"
        autoComplete="new-password"
        value={confirm}
        onChange={onConfirmChange}
        error={mismatch ? "The passwords do not match." : undefined}
      />
    </>
  )
}
