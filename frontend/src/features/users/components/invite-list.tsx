import { useState } from "react"
import { RefreshCwIcon, TicketIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { RoleBadge } from "@/components/common/role-badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { InviteLink } from "@/features/users/components/invite-dialog"
import { formatDate, formatRelativeTime } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useDeleteInvite, useInvites, useRenewInvite } from "@/lib/queries"
import type { CreatedInvite, InviteView } from "@/lib/types"

const label = (i: InviteView) => i.note || (i.username ? `For ${i.username}` : "Username of their choice")

/** Invites that were not used yet; shown only while there are some. Renewing one shows its new link. */
export function InviteList() {
  const invites = useInvites()
  const [renewed, setRenewed] = useState<CreatedInvite | null>(null)
  const renew = useRenewInvite({ onSuccess: setRenewed, onError: failed("renew the invite") })
  const revoke = useDeleteInvite({
    onSuccess: () => toast.success("Invite revoked"),
    onError: failed("revoke the invite"),
  })
  if (!invites.data?.length) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle>Open invites</CardTitle>
        <CardDescription>
          Unused links. Renewing creates a new 7-day link and the old one stops working.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="divide-y rounded-xl border">
          {invites.data.map((i) => {
            return (
              <div key={i.id} className="flex items-center gap-3 p-3">
                <TicketIcon className="size-4 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{label(i)}</div>
                  <div className="text-xs text-muted-foreground">
                    {i.username && i.note && <span className="font-mono">{i.username} · </span>}
                    created {formatRelativeTime(i.createdAt)}
                    {i.createdBy && ` by ${i.createdBy}`} ·{" "}
                    {i.expired ? (
                      <span className="text-destructive">expired</span>
                    ) : (
                      `expires ${formatDate(i.expiresAt)}`
                    )}
                  </div>
                </div>
                <RoleBadge role={i.role} />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Renew ${label(i)}`}
                      disabled={renew.isPending}
                      onClick={() => renew.mutate(i.id)}
                    >
                      <RefreshCwIcon />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>New link, valid for 7 days</TooltipContent>
                </Tooltip>
                <ConfirmDialog
                  trigger={
                    <Button variant="ghost" size="icon-sm" aria-label={`Revoke ${label(i)}`}>
                      <Trash2Icon />
                    </Button>
                  }
                  title="Revoke the invite?"
                  description="The link stops working. This cannot be undone."
                  confirmLabel="Revoke"
                  destructive
                  onConfirm={() => revoke.mutate(i.id)}
                />
              </div>
            )
          })}
        </div>
      </CardContent>
      <Dialog open={!!renewed} onOpenChange={(o) => !o && setRenewed(null)}>
        <DialogContent className="sm:max-w-md">
          {renewed && <InviteLink invite={renewed} onClose={() => setRenewed(null)} />}
        </DialogContent>
      </Dialog>
    </Card>
  )
}
