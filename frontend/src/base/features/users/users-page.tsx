import { useState } from "react"
import { TicketIcon, UserPlusIcon } from "lucide-react"
import { toast } from "sonner"

import { deleteUserNote, usersHint, userTables } from "@/app/users"
import { ConfirmDialog } from "@/base/components/confirm-dialog"
import { QueryState } from "@/base/components/query-state"
import { InviteDialog } from "@/base/features/users/invite-dialog"
import { UserDialog } from "@/base/features/users/user-dialog"
import { UserTable } from "@/base/features/users/user-table"
import { useRetained } from "@/base/hooks/use-retained"
import { PageHeader } from "@/base/layout/page-header"
import { failed, useDeleteUser, useInvites, useUsers } from "@/base/lib/queries"
import type { UserView } from "@/base/lib/types"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"

// One state for all dialogs of the page.
type Dialog = { kind: "edit"; user: UserView | null } | { kind: "delete"; user: UserView } | { kind: "invite" } | null

/**
 * /users (admins): create, invite, edit, disable and delete users, in the tables of the app (app/users.tsx). Invites
 * that were not used yet are rows of the table of the accounts they create.
 */
export function UsersPage() {
  const users = useUsers()
  const invites = useInvites()
  const [dialog, setDialog] = useState<Dialog>(null)
  const remove = useDeleteUser({ onError: failed("delete the user") })
  const deleting = useRetained(dialog?.kind === "delete" ? dialog.user : null)
  const deleteUser = (user: UserView) =>
    remove.mutate(user.username, {
      onSuccess: () => toast.success(`${user.username} deleted`, { description: deleteUserNote(user).done }),
    })
  const actions = {
    onEdit: (user: UserView) => setDialog({ kind: "edit", user }),
    onDelete: (user: UserView) => setDialog({ kind: "delete", user }),
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Users"
        description={usersHint}
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
        {(list) =>
          userTables(list, invites.data ?? []).map(({ title, ...table }) => {
            const card = (
              <Card key={title} className="py-0">
                <UserTable {...table} {...actions} />
              </Card>
            )
            return title ? (
              <section key={title} className="space-y-4">
                <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
                {card}
              </section>
            ) : (
              card
            )
          })
        }
      </QueryState>
      <InviteDialog open={dialog?.kind === "invite"} onClose={() => setDialog(null)} />
      <UserDialog
        open={dialog?.kind === "edit"}
        user={dialog?.kind === "edit" ? dialog.user : null}
        onClose={() => setDialog(null)}
      />
      <ConfirmDialog
        open={dialog?.kind === "delete"}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Delete ${deleting?.username}?`}
        description={deleting && deleteUserNote(deleting).confirm}
        confirmLabel="Delete user"
        destructive
        onConfirm={() => deleting && deleteUser(deleting)}
      />
    </div>
  )
}
