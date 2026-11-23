import { useState } from "react"
import { useSearchParams } from "react-router"
import { Trash2Icon, UserPlusIcon, UsersIcon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/base/components/confirm-dialog"
import { EmptyState, QueryState } from "@/base/components/query-state"
import { UserAvatar } from "@/base/components/user-avatar"
import {
  InviteActions,
  InviteIcon,
  InviteState,
  PasswordResetAction,
} from "@/base/features/users/components/invite-actions"
import { InviteDialog, type InviteExtra } from "@/base/features/users/components/invite-dialog"
import { inviteLabel, splitInvites } from "@/base/features/users/lib/table"
import { useAuth } from "@/base/hooks/use-auth"
import { formatRelativeTime, plural, userName } from "@/base/lib/format"
import { failed, useDeleteUser, useInvites, useSettings, useUsers } from "@/base/lib/queries"
import { type AccessChoice, AccessSelect } from "@/components/common/access-select"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { FieldError, FieldLegend, FieldSet } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { serverName, serverOwner } from "@/lib/format"
import { useServers } from "@/lib/queries"

/**
 * The subusers of the signed-in user (an account card; subusers have none). ?invite=<server> (the Invite button of a
 * server) opens the invite dialog with that server picked.
 */
export function SubusersCard() {
  const { user } = useAuth()
  const [params, setParams] = useSearchParams()
  if (user.owner) return null
  return <Subusers inviteServer={params.get("invite") ?? undefined} onInvited={() => setParams({})} />
}

/**
 * Accounts that see only the servers the signed-in user shares with them, and the invites for more. onInvited is
 * called when the invite dialog closes.
 */
function Subusers({ inviteServer, onInvited }: { inviteServer?: string; onInvited: () => void }) {
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
  const { resets, open } = splitInvites(pending)
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
                          <span>{u.username}</span> · {plural(u.servers, "shared server")} ·{" "}
                          {u.lastLoginAt ? `last login ${formatRelativeTime(u.lastLoginAt)}` : "never signed in"}
                        </div>
                        {reset && (
                          <div className="text-xs">
                            <InviteState invite={reset} />
                          </div>
                        )}
                      </div>
                      <PasswordResetAction username={u.username} reset={reset} />
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
      <InviteDialog open={inviting || !!inviteServer} onClose={closeInvite} extra={subuserInvite(inviteServer)} />
    </Card>
  )
}

/** The invite of a subuser: no role, the servers it sees instead (server is picked at the start). */
const subuserInvite = (server?: string): InviteExtra<Record<string, AccessChoice>> => ({
  title: "Invite a subuser",
  description: "They see only the servers you share with them. The link works once, for 7 days.",
  initial: server ? { [server]: "view" } : {},
  fields: (value, onChange, errors) => (
    <ShareFields
      value={value}
      onChange={onChange}
      error={Object.entries(errors).find(([k]) => k.startsWith("shares"))?.[1]}
    />
  ),
  request: (shares) => ({
    subuser: true,
    shares: Object.entries(shares).flatMap(([s, access]) => (access === "none" ? [] : [{ server: s, access }])),
  }),
})

/** The own servers with the access the subuser gets to each. */
function ShareFields({
  value,
  onChange,
  error,
}: {
  value: Record<string, AccessChoice>
  onChange: (value: Record<string, AccessChoice>) => void
  error?: string
}) {
  const { user } = useAuth()
  const own = useServers().data?.filter((s) => serverOwner(s) === user.username)
  if (!own?.length) return null
  return (
    <FieldSet>
      <FieldLegend variant="label">Servers</FieldLegend>
      <div className="space-y-2">
        {own.map((s) => (
          <div key={s.metadata.name} className="flex items-center gap-3">
            <span className="min-w-0 flex-1 truncate text-sm">{serverName(s)}</span>
            <AccessSelect
              value={value[s.metadata.name] ?? "none"}
              onChange={(access) => onChange({ ...value, [s.metadata.name]: access })}
              label={`Access to ${serverName(s)}`}
            />
          </div>
        ))}
      </div>
      <FieldError>{error}</FieldError>
    </FieldSet>
  )
}
