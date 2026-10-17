import { useState } from "react"
import { TicketIcon, UserPlusIcon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { InviteDialog } from "@/features/users/components/invite-dialog"
import { UserDialog } from "@/features/users/components/user-dialog"
import { UserTable } from "@/features/users/components/user-table"
import { failed } from "@/lib/notify"
import { useDeleteUser, useInvites, useUsers } from "@/lib/queries"
import type { UserView } from "@/lib/types"

// One state for all dialogs of the page.
type Dialog = { kind: "edit"; user: UserView | null } | { kind: "delete"; user: UserView } | { kind: "invite" } | null

/**
 * /users (admins): create, invite, edit, disable and delete users; the subusers of all users in their own table.
 * Invites that were not used yet are rows of the table of the accounts they create.
 */
export function UsersPage() {
  const users = useUsers()
  const invites = useInvites()
  const [dialog, setDialog] = useState<Dialog>(null)
  const remove = useDeleteUser({
    onSuccess: (_, username) =>
      toast.success(`${username} deleted`, { description: "Servers, data and the namespace are being removed." }),
    onError: failed("delete the user"),
  })
  const deleting = dialog?.kind === "delete" ? dialog.user : null

  return (
    <div className="space-y-8">
      <PageHeader
        title="Users"
        actions={
          <>
            <Button variant="outline" onClick={() => setDialog({ kind: "invite" })}>
              <TicketIcon />
              Invite
            </Button>
            <Button onClick={() => setDialog({ kind: "edit", user: null })}>
              <UserPlusIcon />
              New user
            </Button>
          </>
        }
      />
      <QueryState query={users}>
        {(list) => {
          // Reset links belong to an account of either table; the rows find them by username.
          const pending = invites.data ?? []
          const subuserInvites = pending.filter((i) => i.reset || i.owner)
          const subusers = list.filter((u) => u.owner)
          const actions = {
            onEdit: (user: UserView) => setDialog({ kind: "edit", user }),
            onDelete: (user: UserView) => setDialog({ kind: "delete", user }),
          }
          return (
            <>
              <Card className="py-0">
                <UserTable
                  users={list.filter((u) => !u.owner)}
                  invites={pending.filter((i) => i.reset || !i.owner)}
                  {...actions}
                />
              </Card>
              {(subusers.length > 0 || subuserInvites.some((i) => !i.reset)) && (
                <section className="space-y-4">
                  <h2 className="text-lg font-semibold tracking-tight">Subusers</h2>
                  <Card className="py-0">
                    <UserTable users={subusers} invites={subuserInvites} subusers {...actions} />
                  </Card>
                </section>
              )}
            </>
          )
        }}
      </QueryState>
      <InviteDialog open={dialog?.kind === "invite"} onClose={() => setDialog(null)} />
      <UserDialog
        open={dialog?.kind === "edit"}
        user={dialog?.kind === "edit" ? dialog.user : null}
        onClose={() => setDialog(null)}
      />
      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Delete ${deleting?.username}?`}
        description={
          deleting?.owner ? (
            "The account is deleted permanently and loses access to the servers shared with it."
          ) : (
            <>
              All {deleting?.servers ?? 0} game server(s) of this user, their data, the namespace{" "}
              <code className="font-mono">{deleting?.namespace}</code> and their subusers are deleted permanently.
            </>
          )
        }
        confirmLabel="Delete user"
        destructive
        onConfirm={() => deleting && remove.mutate(deleting.username)}
      />
    </div>
  )
}
