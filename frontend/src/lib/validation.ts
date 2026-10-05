import { ApiError } from "@/lib/api"

/** Same minimum as the panel (auth.HashPassword). */
export const MIN_PASSWORD_LENGTH = 8

/** Whether a new password is long enough and typed the same twice. */
export const newPasswordReady = (password: string, confirm: string) =>
  password.length >= MIN_PASSWORD_LENGTH && password === confirm

/**
 * Messages of a failed mutation for the form: per field for validation errors (HTTP 422
 * "fields"), anything else under "form". Shown with <FieldError> next to the input.
 */
export function fieldErrors(err: Error | null): Record<string, string> {
  if (!err) return {}
  if (err instanceof ApiError && err.fields) return err.fields
  return { form: err.message }
}
