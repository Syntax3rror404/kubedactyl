import { useState } from "react"
import { Navigate, useNavigate, useSearchParams } from "react-router"
import { RocketIcon, ShieldCheckIcon } from "lucide-react"
import { toast } from "sonner"

import { appName, envPrefix, setupNext } from "@/app/config"
import { AuthShell } from "@/base/features/auth/auth-shell"
import { NewAccountFields } from "@/base/features/auth/new-account-fields"
import { NewPasswordFields } from "@/base/features/auth/new-password-fields"
import { useDraft } from "@/base/hooks/use-draft"
import { ApiError } from "@/base/lib/api"
import { useRunSetup, useSetupStatus } from "@/base/lib/queries"
import { fieldErrors, newPasswordReady } from "@/base/lib/validation"
import { Button } from "@/components/ui/button"
import { CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"

/** First start: create the first administrator (only reachable while none exists). */
export function SetupPage() {
  const status = useSetupStatus()
  const [params] = useSearchParams()
  // The one-time link from the panel log carries the token; otherwise it is typed in.
  const linkToken = params.get("token") ?? ""
  const navigate = useNavigate()
  const { draft, set } = useDraft({ token: linkToken, username: "admin", displayName: "", password: "", confirm: "" })
  // Set on success, so the "already set up" redirect does not race the navigation.
  const [done, setDone] = useState(false)
  const setupAdmin = useRunSetup({
    onSuccess: () => {
      setDone(true)
      toast.success(`${appName} is ready`, { description: setupNext.hint })
      navigate(setupNext.to, { replace: true })
    },
    // Someone else finished the setup meanwhile.
    onError: (err) => err instanceof ApiError && err.status === 409 && void status.refetch(),
  })
  const errors = fieldErrors(setupAdmin.error)

  if (status.isLoading)
    return (
      <AuthShell wide title="Setup">
        <CardContent className="flex justify-center py-6">
          <Spinner className="size-6" />
        </CardContent>
      </AuthShell>
    )
  if (!done && status.data && !status.data.required) return <Navigate to="/login" replace />

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    setupAdmin.mutate({
      username: draft.username,
      password: draft.password,
      displayName: draft.displayName || undefined,
      token: draft.token.trim(),
    })
  }

  return (
    <AuthShell wide title="Setup">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <RocketIcon className="size-4" />
          Set up {appName}
        </CardTitle>
        <CardDescription>Create the first administrator.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit}>
          <FieldGroup>
            {(!linkToken || errors.token) && (
              <Field data-invalid={!!errors.token}>
                <FieldLabel htmlFor="token">Setup token</FieldLabel>
                <Input
                  id="token"
                  autoFocus
                  value={draft.token}
                  onChange={(e) => set("token", e.target.value)}
                  className="font-mono"
                  aria-invalid={!!errors.token}
                />
                {errors.token ? (
                  <FieldError>{errors.token}</FieldError>
                ) : (
                  <FieldDescription>
                    Part of the setup link in the panel log, or {envPrefix}_SETUP_TOKEN.
                  </FieldDescription>
                )}
              </Field>
            )}
            <NewAccountFields
              username={draft.username}
              displayName={draft.displayName}
              onUsernameChange={(v) => set("username", v)}
              onDisplayNameChange={(v) => set("displayName", v)}
              error={errors.username}
              displayPlaceholder="Administrator"
            />
            <NewPasswordFields
              password={draft.password}
              confirm={draft.confirm}
              onPasswordChange={(v) => set("password", v)}
              onConfirmChange={(v) => set("confirm", v)}
              error={errors.password}
              autoFocus={!!linkToken}
            />
            {errors.form && <FieldError>{errors.form}</FieldError>}
            <Button
              type="submit"
              className="w-full"
              disabled={
                setupAdmin.isPending ||
                !draft.username ||
                !draft.token.trim() ||
                !newPasswordReady(draft.password, draft.confirm)
              }
            >
              {setupAdmin.isPending ? <Spinner /> : <ShieldCheckIcon />}
              Create administrator
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
    </AuthShell>
  )
}
