import { useState } from "react"
import { Trash2Icon, UserPlusIcon, UsersIcon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { EmptyState, QueryState } from "@/components/common/query-state"
import { UserAvatar } from "@/components/common/user-avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { InviteActions, InviteIcon, InviteState, ResetPasswordButton } from "@/features/users/components/invite-actions"
import { InviteDialog } from "@/features/users/components/invite-dialog"
import { inviteLabel } from "@/features/users/lib/invite"
import { useAuth } from "@/hooks/use-auth"
import { formatRelativeTime, plural, userName } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useDeleteUser, useInvites, useSettings, useUsers } from "@/lib/queries"

/**
 * Accounts that see only the servers the signed-in user shares with them, and the invites for more. inviteServer
 * (the Invite button of a server) opens the invite dialog with that server picked; onInvited is called when it closes.
 */
export function SubusersCard({ inviteServer, onInvited }: { inviteServer?: string; onInvited: () => void }) {
  const { user: me } = useAuth()
  const users = useUsers()
  const invites = useInvites()
  const limit = useSettings().data?.subusersPerUser ?? 10
  const [inviting, setInviting] = useState(false)
  const remove = useDeleteUser({
    onSuccess: (_, username) => toast.success(`${username} deleted`),
    onError: failed("delete the subuser"),
  })
  const mine = users.data?.filter((u) => u.owner === me.username) ?? []
  const pending = invites.data?.filter((i) => i.owner === me.username) ?? []
  const resets = new Map(pending.filter((i) => i.reset).map((i) => [i.username, i]))
  const open = pending.filter((i) => !i.reset)
  const used = mine.length + open.length
  const closeInvite = () => {
    setInviting(false)
    onInvited()
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Subusers</CardTitle>
        <CardDescription>
          {limit > 0
            ? `Friends who may watch or control the servers you share with them (${used} of ${limit} used).`
            : "Subusers are turned off by the administrator."}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {limit > 0 && (
          <Button variant="outline" disabled={used >= limit} onClick={() => setInviting(true)}>
            <UserPlusIcon />
            Invite a subuser
          </Button>
        )}
        <QueryState query={users} skeleton={<Skeleton className="h-16 rounded-xl" />}>
          {() =>
            used === 0 ? (
              <EmptyState variant="card" icon={<UsersIcon />} title="No subusers yet" />
            ) : (
              <div className="divide-y rounded-xl border">
                {mine.map((u) => {
                  const reset = resets.get(u.username)
                  return (
                    <div key={u.username} className="flex items-center gap-3 p-3">
                      <UserAvatar user={u} />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{userName(u)}</div>
                        <div className="text-xs text-muted-foreground">
                          <span className="font-mono">{u.username}</span> · {plural(u.servers, "shared server")} ·{" "}
                          {u.lastLoginAt ? `last login ${formatRelativeTime(u.lastLoginAt)}` : "never signed in"}
                        </div>
                        {reset && (
                          <div className="text-xs">
                            <InviteState invite={reset} />
                          </div>
                        )}
                      </div>
                      {reset ? <InviteActions invite={reset} /> : <ResetPasswordButton username={u.username} />}
                      <ConfirmDialog
                        trigger={
                          <Button variant="ghost" size="icon-sm" aria-label={`Delete ${u.username}`}>
                            <Trash2Icon />
                          </Button>
                        }
                        title={`Delete ${u.username}?`}
                        description="The account is deleted and loses access to your servers. This cannot be undone."
                        confirmLabel="Delete"
                        destructive
                        onConfirm={() => remove.mutate(u.username)}
                      />
                    </div>
                  )
                })}
                {open.map((i) => (
                  <div key={i.id} className="flex items-center gap-3 p-3">
                    <InviteIcon />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{inviteLabel(i)}</div>
                      <div className="text-xs">
                        <span className="text-muted-foreground">{plural(i.shares.length, "shared server")} · </span>
                        <InviteState invite={i} />
                      </div>
                    </div>
                    <InviteActions invite={i} />
                  </div>
                ))}
              </div>
            )
          }
        </QueryState>
      </CardContent>
      <InviteDialog open={inviting || !!inviteServer} onClose={closeInvite} subuser server={inviteServer} />
    </Card>
  )
}
