import { KeyRoundIcon, LogOutIcon } from "lucide-react"
import { Navigate, useNavigate } from "react-router"
import { toast } from "sonner"

import { UnveilPassword } from "@/components/common/unveil-password"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { NewPasswordFields } from "@/features/auth/components/new-password-fields"
import { useAuth } from "@/hooks/use-auth"
import { useDraft } from "@/hooks/use-draft"
import { useLogout, useUpdatePassword } from "@/lib/queries"
import { fieldErrors, newPasswordReady } from "@/lib/validation"

/** /change-password: a user whose password an administrator set chooses an own one before anything else. */
export function ChangePasswordPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const { draft, set } = useDraft({ current: "", password: "", confirm: "" })
  // Changing the password ends every session; the user signs in with the new one.
  const change = useUpdatePassword({
    onSuccess: () => {
      toast.success("Password changed", { description: "Sign in with your new password." })
      navigate("/login", { replace: true })
    },
  })
  const logout = useLogout({ onSettled: () => navigate("/login", { replace: true }) })
  const errors = fieldErrors(change.error)

  if (!user.mustChangePassword) return <Navigate to="/" replace />

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    change.mutate({ current: draft.current, next: draft.password, revokeTokens: false })
  }

  return (
    <AuthShell title="New password">
      <Card className="bg-card/80 backdrop-blur">
        <CardHeader>
          <CardTitle>Choose your password</CardTitle>
          <CardDescription>
            Your administrator set a start password. Replace it with your own to continue.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit}>
            <FieldGroup>
              <Field data-invalid={!!errors.current}>
                <FieldLabel htmlFor="current">Start password</FieldLabel>
                <UnveilPassword
                  id="current"
                  autoComplete="current-password"
                  autoFocus
                  value={draft.current}
                  onChange={(e) => set("current", e.target.value)}
                  aria-invalid={!!errors.current}
                />
                {errors.current && <FieldError>{errors.current}</FieldError>}
              </Field>
              <NewPasswordFields
                password={draft.password}
                confirm={draft.confirm}
                onPassword={(v) => set("password", v)}
                onConfirm={(v) => set("confirm", v)}
                error={errors.new}
                label="New password"
              />
              {errors.form && <FieldError>{errors.form}</FieldError>}
              <Button
                type="submit"
                className="w-full"
                disabled={change.isPending || !draft.current || !newPasswordReady(draft.password, draft.confirm)}
              >
                {change.isPending ? <Spinner /> : <KeyRoundIcon />}
                Save password
              </Button>
              <Button type="button" variant="ghost" className="w-full" onClick={() => logout.mutate()}>
                <LogOutIcon />
                Sign out
              </Button>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </AuthShell>
  )
}
