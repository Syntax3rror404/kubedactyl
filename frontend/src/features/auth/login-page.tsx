import { KeyRoundIcon, LogInIcon } from "lucide-react"
import { useEffect, useRef, useState } from "react"
import { Navigate, useNavigate, useSearchParams } from "react-router"

import { UnveilPassword } from "@/components/common/unveil-password"
import { VerifyMark, verifyFailMs, verifyOkMs, type VerifyState } from "@/components/common/verify-mark"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldError, FieldGroup, FieldLabel, FieldSeparator } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { useDraft } from "@/hooks/use-draft"
import { urls } from "@/lib/api"
import { useLogin, useMe, useOIDCSignIn, useSetupStatus } from "@/lib/queries"
import { cn } from "@/lib/utils"
import { fieldErrors } from "@/lib/validation"

/** Why a sign-in through the identity provider failed (/login?sso=<reason>, set by the panel). */
const ssoErrors: Record<string, string> = {
  access: "Your account has no access to this panel.",
  account: "This account cannot sign in here. Ask your administrator.",
  failed: "Single sign-on failed. Please try again.",
}

type SignInVia = "password" | "sso"
type Answer = "ok" | "fail"

// The ticks turn at least this long, so a fast answer (or one already known, back from the identity provider)
// still shows the wait and the ring closing.
const minWaitMs = 500

/**
 * What the sign-in page shows instead of the form: waiting for the panel (password) or the identity provider
 * (SSO), then the answer. Accepted opens `target`, rejected brings the form back. `sso` is the result the panel
 * sent after a sign-in through the identity provider (/login?sso=ok|<reason>).
 */
function useSignInFlow(target: string, sso: string | null) {
  const navigate = useNavigate()
  const [via, setVia] = useState<SignInVia | null>(sso ? "sso" : null)
  // The answer as it came and as it shows (after the shortest wait).
  const [result, setResult] = useState<Answer | null>(sso ? (sso === "ok" ? "ok" : "fail") : null)
  const [shown, setShown] = useState<Answer | null>(null)
  const waitingSince = useRef(0)

  const reset = () => {
    setVia(null)
    setResult(null)
    setShown(null)
  }

  // Back from the identity provider the page waits from its start.
  useEffect(() => {
    waitingSince.current = performance.now()
  }, [])

  useEffect(() => {
    if (!result) return
    const timer = setTimeout(() => setShown(result), minWaitMs - (performance.now() - waitingSince.current))
    return () => clearTimeout(timer)
  }, [result])

  // The panel opens or the form comes back as soon as the check or cross has finished.
  useEffect(() => {
    if (!shown) return
    const timer = setTimeout(
      () => (shown === "ok" ? navigate(target, { replace: true }) : reset()),
      shown === "ok" ? verifyOkMs : verifyFailMs,
    )
    return () => clearTimeout(timer)
  }, [shown, navigate, target])

  // Back from the identity provider's page with the browser's back button: the page comes from the cache, still
  // waiting.
  useEffect(() => {
    const restored = (e: PageTransitionEvent) => e.persisted && reset()
    window.addEventListener("pageshow", restored)
    return () => window.removeEventListener("pageshow", restored)
  }, [])

  const state: VerifyState | null = shown ?? (via ? "waiting" : null)
  return {
    via,
    state,
    wait: (next: SignInVia) => {
      waitingSince.current = performance.now()
      setVia(next)
      setResult(null)
      setShown(null)
    },
    answer: (ok: boolean) => setResult(ok ? "ok" : "fail"),
  }
}

// What the card says while the panel or the identity provider answers and after it answered.
const progressTexts: Record<VerifyState, Record<SignInVia, string>> = {
  waiting: { password: "Verifying...", sso: "Contacting provider..." },
  ok: { password: "Accepted", sso: "Accepted" },
  fail: { password: "Rejected", sso: "Rejected" },
}

/** The verify mark of a sign-in with one line under it. */
function SignInProgress({ via, state }: { via: SignInVia; state: VerifyState }) {
  const text = progressTexts[state][via]
  return (
    <div className="flex flex-col items-center justify-center gap-4 text-center" aria-live="polite">
      <VerifyMark state={state} label={text} />
      <span className="font-medium">{text}</span>
    </div>
  )
}

/**
 * /login: username and password, and single sign-on when it is on; redirects to /setup while no administrator
 * exists. While the panel or the identity provider answers, the form gives way to the verify mark.
 */
export function LoginPage() {
  const [params] = useSearchParams()
  const next = params.get("next") || "/"
  const target = next.startsWith("/") ? next : "/"
  const flow = useSignInFlow(target, params.get("sso"))
  const login = useLogin({ onSuccess: () => flow.answer(true), onError: () => flow.answer(false) })
  const me = useMe()
  const setup = useSetupStatus()
  const sso = useOIDCSignIn()
  const ssoError = ssoErrors[params.get("sso") ?? ""]
  const { draft, set } = useDraft({ username: "", password: "" })
  const error = fieldErrors(login.error).form

  // Not while the answer of a sign-in shows; the flow opens the target itself.
  if (me.data && !flow.state) return <Navigate to={target} replace />
  if (setup.data?.required) return <Navigate to="/setup" replace />

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    flow.wait("password")
    login.mutate(draft)
  }

  return (
    <AuthShell title="Sign in">
      <Card className="bg-card/80 backdrop-blur">
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
        </CardHeader>
        {/* Form and progress share one cell, so the card keeps its height when they swap. */}
        <CardContent className="grid *:col-start-1 *:row-start-1">
          <form
            onSubmit={submit}
            inert={!!flow.state}
            className={cn("transition-opacity duration-300", flow.state && "opacity-0")}
          >
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
              <Button type="submit" className="w-full" disabled={!draft.username || !draft.password}>
                <LogInIcon />
                Sign in
              </Button>
              {sso.data?.enabled && (
                <>
                  <FieldSeparator />
                  <Field>
                    <Button variant="outline" className="w-full" asChild>
                      <a href={urls.oidcStart(target)} onClick={() => flow.wait("sso")}>
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
          {flow.via && flow.state && <SignInProgress via={flow.via} state={flow.state} />}
        </CardContent>
      </Card>
    </AuthShell>
  )
}
