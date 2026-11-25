import { Link, useNavigate, useSearchParams } from "react-router"
import { KeyRoundIcon, TicketIcon, UserPlusIcon } from "lucide-react"
import { toast } from "sonner"

import { AuthShell } from "@/base/features/auth/auth-shell"
import { NewAccountFields } from "@/base/features/auth/new-account-fields"
import { NewPasswordFields } from "@/base/features/auth/new-password-fields"
import { useDraft } from "@/base/hooks/use-draft"
import { formatDate } from "@/base/lib/format"
import { useAcceptInvite, useInvite } from "@/base/lib/queries"
import type { InviteDetails } from "@/base/lib/types"
import { fieldErrors, newPasswordReady } from "@/base/lib/validation"
import { Button } from "@/components/ui/button"
import { CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { FieldError, FieldGroup } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"

/** /invite?token=…: creates an account with an invite link, or sets a new password with a reset link. */
export function InvitePage() {
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const invite = useInvite(token)

  return (
    <AuthShell wide title="Invite">
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
            The link was used already, has expired or was revoked. Ask for a new one. If you already have an account,{" "}
            <Link to="/login" className="underline underline-offset-4">
              sign in
            </Link>
            .
          </CardDescription>
        </CardHeader>
      )}
    </AuthShell>
  )
}

function AccountForm({ token, invite }: { token: string; invite: InviteDetails }) {
  const navigate = useNavigate()
  const { draft, set } = useDraft({ username: invite.username ?? "", displayName: "", password: "", confirm: "" })
  const accept = useAcceptInvite({
    onSuccess: (res) => {
      toast.success(invite.reset ? "Password set" : `Welcome, ${res.user.displayName || res.user.username}`)
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
          {invite.reset ? <KeyRoundIcon className="size-4" /> : <TicketIcon className="size-4" />}
          {invite.reset ? `New password for ${invite.username}` : "Create your account"}
        </CardTitle>
        <CardDescription>
          {invite.reset ? "Every other sign-in of the account ends." : "You were invited."} The link works once, until{" "}
          {formatDate(invite.expiresAt)}.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit}>
          <FieldGroup>
            {!invite.reset && (
              <NewAccountFields
                username={draft.username}
                displayName={draft.displayName}
                onUsernameChange={(v) => set("username", v)}
                onDisplayNameChange={(v) => set("displayName", v)}
                error={errors.username}
                fixedUsername={!!invite.username}
                autoFocus
              />
            )}
            <NewPasswordFields
              password={draft.password}
              confirm={draft.confirm}
              onPasswordChange={(v) => set("password", v)}
              onConfirmChange={(v) => set("confirm", v)}
              error={errors.password}
            />
            {errors.form && <FieldError>{errors.form}</FieldError>}
            <Button
              type="submit"
              className="w-full"
              disabled={accept.isPending || !draft.username || !newPasswordReady(draft.password, draft.confirm)}
            >
              {accept.isPending ? <Spinner /> : invite.reset ? <KeyRoundIcon /> : <UserPlusIcon />}
              {invite.reset ? "Set password" : "Create account"}
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
    </>
  )
}
