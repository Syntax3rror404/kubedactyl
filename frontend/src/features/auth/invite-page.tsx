import { TicketIcon, UserPlusIcon } from "lucide-react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { NewPasswordFields } from "@/features/auth/components/new-password-fields"
import { useDraft } from "@/hooks/use-draft"
import { formatDate } from "@/lib/format"
import { useAcceptInvite, useInvite } from "@/lib/queries"
import type { InviteDetails } from "@/lib/types"
import { fieldErrors, newPasswordReady } from "@/lib/validation"

/** /invite?token=…: creates an account with an invite link from an administrator. */
export function InvitePage() {
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const invite = useInvite(token)

  return (
    <AuthShell wide title="Invite">
      <Card className="bg-card/80 backdrop-blur">
        {invite.data ? (
          <AccountForm token={token} invite={invite.data} />
        ) : invite.isLoading ? (
          <CardContent className="flex justify-center py-6">
            <Spinner className="size-6" />
          </CardContent>
        ) : (
          <CardHeader>
            <CardTitle>Invite link not valid</CardTitle>
            <CardDescription>
              The link was used already, has expired or was revoked. Ask your administrator for a new one. If you
              already have an account,{" "}
              <Link to="/login" className="underline underline-offset-4">
                sign in
              </Link>
              .
            </CardDescription>
          </CardHeader>
        )}
      </Card>
    </AuthShell>
  )
}

function AccountForm({ token, invite }: { token: string; invite: InviteDetails }) {
  const navigate = useNavigate()
  const { draft, set } = useDraft({ username: invite.username ?? "", displayName: "", password: "", confirm: "" })
  const accept = useAcceptInvite({
    onSuccess: (res) => {
      toast.success(`Welcome, ${res.user.displayName || res.user.username}`)
      navigate("/", { replace: true })
    },
  })
  const errors = fieldErrors(accept.error)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    accept.mutate({
      token,
      username: draft.username,
      displayName: draft.displayName || undefined,
      password: draft.password,
    })
  }

  return (
    <>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TicketIcon className="size-4" />
          Create your account
        </CardTitle>
        <CardDescription>You were invited. The link works once, until {formatDate(invite.expiresAt)}.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit}>
          <FieldGroup>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field data-invalid={!!errors.username}>
                <FieldLabel htmlFor="username">Username</FieldLabel>
                <Input
                  id="username"
                  autoComplete="username"
                  autoFocus={!invite.username}
                  readOnly={!!invite.username}
                  value={draft.username}
                  onChange={(e) => set("username", e.target.value.toLowerCase())}
                  className="font-mono"
                  aria-invalid={!!errors.username}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="display">Display name</FieldLabel>
                <Input
                  id="display"
                  autoFocus={!!invite.username}
                  value={draft.displayName}
                  onChange={(e) => set("displayName", e.target.value)}
                  placeholder="optional"
                />
              </Field>
            </div>
            {errors.username ? (
              <FieldError>{errors.username}</FieldError>
            ) : (
              <FieldDescription>
                {invite.username
                  ? "Your administrator chose the username."
                  : "3-32 lowercase letters, digits or dashes, starting with a letter. It cannot be changed later."}
              </FieldDescription>
            )}
            <NewPasswordFields
              password={draft.password}
              confirm={draft.confirm}
              onPassword={(v) => set("password", v)}
              onConfirm={(v) => set("confirm", v)}
              error={errors.password}
            />
            {errors.form && <FieldError>{errors.form}</FieldError>}
            <Button
              type="submit"
              className="w-full"
              disabled={accept.isPending || !draft.username || !newPasswordReady(draft.password, draft.confirm)}
            >
              {accept.isPending ? <Spinner /> : <UserPlusIcon />}
              Create account
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
    </>
  )
}
