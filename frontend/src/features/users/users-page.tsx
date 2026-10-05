import { useState } from "react"
import { TicketIcon, UserPlusIcon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { InviteDialog } from "@/features/users/components/invite-dialog"
import { InviteList } from "@/features/users/components/invite-list"
import { UserDialog } from "@/features/users/components/user-dialog"
import { UserTable } from "@/features/users/components/user-table"
import { failed } from "@/lib/notify"
import { useDeleteUser, useUsers } from "@/lib/queries"
import type { UserView } from "@/lib/types"

// One state for all dialogs of the page.
type Dialog = { kind: "edit"; user: UserView | null } | { kind: "delete"; user: UserView } | { kind: "invite" } | null

/** /users (admins): create, invite, edit, disable and delete users. */
export function UsersPage() {
  const users = useUsers()
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
        {(list) => (
          <Card className="py-0">
            <UserTable
              users={list}
              onEdit={(user) => setDialog({ kind: "edit", user })}
              onDelete={(user) => setDialog({ kind: "delete", user })}
            />
          </Card>
        )}
      </QueryState>
      <InviteList />
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
          <>
            All {deleting?.servers ?? 0} game server(s) of this user, their data and the namespace{" "}
            <code className="font-mono">{deleting?.namespace}</code> are deleted permanently.
          </>
        }
        confirmLabel="Delete user"
        destructive
        onConfirm={() => deleting && remove.mutate(deleting.username)}
      />
    </div>
  )
}
