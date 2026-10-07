import { KeyRoundIcon, LogInIcon } from "lucide-react"
import { Navigate, useNavigate, useSearchParams } from "react-router"

import { UnveilPassword } from "@/components/common/unveil-password"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldError, FieldGroup, FieldLabel, FieldSeparator } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { useDraft } from "@/hooks/use-draft"
import { urls } from "@/lib/api"
import { useLogin, useMe, useOIDCSignIn, useSetupStatus } from "@/lib/queries"
import { fieldErrors } from "@/lib/validation"

/** Why a sign-in through the identity provider failed (/login?sso=<reason>, set by the panel). */
const ssoErrors: Record<string, string> = {
  access: "Your account has no access to this panel.",
  account: "This account cannot sign in here. Ask your administrator.",
  failed: "Single sign-on failed. Please try again.",
}

/**
 * /login: username and password, and single sign-on when it is on; redirects to /setup while no administrator
 * exists.
 */
export function LoginPage() {
  const [params] = useSearchParams()
  const next = params.get("next") || "/"
  const target = next.startsWith("/") ? next : "/"
  const navigate = useNavigate()
  const login = useLogin({ onSuccess: () => navigate(target, { replace: true }) })
  const me = useMe()
  const setup = useSetupStatus()
  const sso = useOIDCSignIn()
  const ssoError = ssoErrors[params.get("sso") ?? ""]
  const { draft, set } = useDraft({ username: "", password: "" })
  const error = fieldErrors(login.error).form

  if (me.data) return <Navigate to={target} replace />
  if (setup.data?.required) return <Navigate to="/setup" replace />

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    login.mutate(draft)
  }

  return (
    <AuthShell title="Sign in">
      <Card className="bg-card/80 backdrop-blur">
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="username">Username</FieldLabel>
                <Input
                  id="username"
                  autoComplete="username"
                  autoFocus
                  value={draft.username}
                  onChange={(e) => set("username", e.target.value)}
                />
              </Field>
              <Field data-invalid={!!error}>
                <FieldLabel htmlFor="password">Password</FieldLabel>
                <UnveilPassword
                  id="password"
                  autoComplete="current-password"
                  value={draft.password}
                  onChange={(e) => set("password", e.target.value)}
                  aria-invalid={!!error}
                />
                {error && <FieldError>{error}</FieldError>}
              </Field>
              <Button type="submit" className="w-full" disabled={login.isPending || !draft.username || !draft.password}>
                {login.isPending ? <Spinner /> : <LogInIcon />}
                Sign in
              </Button>
              {sso.data?.enabled && (
                <>
                  <FieldSeparator />
                  <Field>
                    <Button variant="outline" className="w-full" asChild>
                      <a href={urls.oidcStart(target)}>
                        <KeyRoundIcon />
                        Sign in with {sso.data.name}
                      </a>
                    </Button>
                    {ssoError && <FieldError>{ssoError}</FieldError>}
                  </Field>
                </>
              )}
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </AuthShell>
  )
}
