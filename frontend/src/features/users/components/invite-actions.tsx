import { useState } from "react"
import { KeyRoundIcon, RefreshCwIcon, TicketIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { InviteLinkDialog } from "@/features/users/components/invite-dialog"
import { inviteLabel } from "@/features/users/lib/invite"
import { formatDate } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useCreateInvite, useDeleteInvite, useRenewInvite } from "@/lib/queries"
import type { CreatedInvite, InviteView } from "@/lib/types"

/** Stands in for the avatar of an account an invite has not created yet. */
export function InviteIcon() {
  return (
    <div className="flex size-8 shrink-0 items-center justify-center rounded-lg border border-dashed">
      <TicketIcon className="size-4 text-muted-foreground" />
    </div>
  )
}

/** State of a pending invite or reset link for the row it is shown in. */
export function InviteState({ invite }: { invite: InviteView }) {
  const what = invite.reset ? "Password reset pending" : "Invite pending"
  return invite.expired ? (
    <span className="text-destructive">{what}, expired</span>
  ) : (
    <span className="text-amber-600 dark:text-amber-400">
      {what} until {formatDate(invite.expiresAt)}
    </span>
  )
}

/** Renew (a new 7-day link, shown once) and revoke buttons of a pending invite or reset link. */
export function InviteActions({ invite }: { invite: InviteView }) {
  const [renewed, setRenewed] = useState<CreatedInvite | null>(null)
  const renew = useRenewInvite({ onSuccess: setRenewed, onError: failed("renew the link") })
  const revoke = useDeleteInvite({
    onSuccess: () => toast.success(invite.reset ? "Reset link revoked" : "Invite revoked"),
    onError: failed("revoke the link"),
  })
  const label = invite.reset ? `reset link of ${invite.username}` : `invite ${inviteLabel(invite)}`
  return (
    <div className="flex items-center">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={`Renew ${label}`}
            disabled={renew.isPending}
            onClick={() => renew.mutate(invite.id)}
          >
            <RefreshCwIcon />
          </Button>
        </TooltipTrigger>
        <TooltipContent>New link, valid for 7 days</TooltipContent>
      </Tooltip>
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon-sm" aria-label={`Revoke ${label}`}>
            <Trash2Icon />
          </Button>
        }
        title={invite.reset ? "Revoke the reset link?" : "Revoke the invite?"}
        description="The link stops working. This cannot be undone."
        confirmLabel="Revoke"
        destructive
        onConfirm={() => revoke.mutate(invite.id)}
      />
      <InviteLinkDialog invite={renewed} onClose={() => setRenewed(null)} />
    </div>
  )
}

/** Creates a link with which a local account sets a new password, and shows it once. */
export function ResetPasswordButton({ username }: { username: string }) {
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const create = useCreateInvite({ onSuccess: setCreated, onError: failed("create the reset link") })
  return (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={`Reset the password of ${username}`}
            disabled={create.isPending}
            onClick={() => create.mutate({ username, reset: true })}
          >
            <KeyRoundIcon />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Password reset link</TooltipContent>
      </Tooltip>
      <InviteLinkDialog invite={created} onClose={() => setCreated(null)} />
    </>
  )
}
