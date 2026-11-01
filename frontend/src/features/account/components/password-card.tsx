import { useState } from "react"
import { KeyRoundIcon } from "lucide-react"
import { useNavigate } from "react-router"
import { toast } from "sonner"

import { PasswordField } from "@/components/common/unveil-password"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { NewPasswordFields } from "@/features/auth/components/new-password-fields"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useUpdatePassword } from "@/lib/queries"
import { fieldErrors, newPasswordReady } from "@/lib/validation"

/**
 * Change the own password; this signs the user out everywhere, including this browser. The API tokens are revoked,
 * too, unless the user keeps them (a routine change, not a leak).
 */
export function PasswordCard() {
  const navigate = useNavigate()
  const { draft, set } = useDraft({ current: "", password: "", confirm: "" })
  const [revokeTokens, setRevokeTokens] = useState(true)
  const change = useUpdatePassword({
    onSuccess: () => {
      toast.success("Password changed", { description: "Please sign in again." })
      navigate("/login", { replace: true })
    },
    onError: failed("change the password"),
  })
  const errors = fieldErrors(change.error)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    change.mutate({ current: draft.current, next: draft.password, revokeTokens })
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Password</CardTitle>
        <CardDescription>Changing the password signs you out on all devices.</CardDescription>
      </CardHeader>
      <form onSubmit={submit} className="contents">
        <CardContent>
          <FieldGroup>
            <PasswordField
              id="pw-current"
              label="Current password"
              autoComplete="current-password"
              value={draft.current}
              onChange={(v) => set("current", v)}
              error={errors.current}
            />
            <NewPasswordFields
              password={draft.password}
              confirm={draft.confirm}
              onPasswordChange={(v) => set("password", v)}
              onConfirmChange={(v) => set("confirm", v)}
              error={errors.new}
              label="New password"
            />
            <Field orientation="horizontal">
              <Switch id="pw-revoke-tokens" checked={revokeTokens} onCheckedChange={setRevokeTokens} />
              <FieldLabel htmlFor="pw-revoke-tokens">Also revoke my API tokens</FieldLabel>
            </Field>
          </FieldGroup>
        </CardContent>
        <CardFooter className="justify-end">
          <Button
            type="submit"
            disabled={change.isPending || !draft.current || !newPasswordReady(draft.password, draft.confirm)}
          >
            {change.isPending ? <Spinner /> : <KeyRoundIcon />}
            Change password
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}
