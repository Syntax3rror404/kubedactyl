import { MoreHorizontalIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { OidcBadge, RoleBadge } from "@/components/common/role-badge"
import { UserAvatar } from "@/components/common/user-avatar"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { InviteActions, InviteIcon, InviteState, ResetPasswordButton } from "@/features/users/components/invite-actions"
import { inviteLabel } from "@/features/users/lib/invite"
import { useAuth } from "@/hooks/use-auth"
import { formatRelativeTime, plural, roleLabel, userName } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useUpdateUser } from "@/lib/queries"
import type { InviteView, UserView } from "@/lib/types"

/**
 * Users with role, state, servers and actions, followed by the invites that were not used yet. Subusers show their
 * owner instead of role and namespace. Local accounts get a password reset link; a pending one is shown in the row.
 */
export function UserTable({
  users,
  invites,
  subusers,
  onEdit,
  onDelete,
}: {
  users: UserView[]
  invites: InviteView[]
  subusers?: boolean
  onEdit: (u: UserView) => void
  onDelete: (u: UserView) => void
}) {
  const resets = new Map(invites.filter((i) => i.reset).map((i) => [i.username, i]))
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="pl-4">User</TableHead>
          {subusers ? (
            <TableHead className="hidden sm:table-cell">Owner</TableHead>
          ) : (
            <>
              <TableHead className="hidden sm:table-cell">Role</TableHead>
              <TableHead className="hidden md:table-cell">K8s Namespace</TableHead>
            </>
          )}
          <TableHead className="hidden text-right sm:table-cell">{subusers ? "Shared servers" : "Servers"}</TableHead>
          <TableHead className="hidden text-right lg:table-cell">Last login</TableHead>
          <TableHead className="text-center">Active</TableHead>
          <TableHead className="w-12" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((u) => (
          <UserRow
            key={u.username}
            user={u}
            reset={resets.get(u.username)}
            subusers={subusers}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
        {invites
          .filter((i) => !i.reset)
          .map((i) => (
            <InviteRow key={i.id} invite={i} subusers={subusers} />
          ))}
      </TableBody>
    </Table>
  )
}

function UserRow({
  user: u,
  reset,
  subusers,
  onEdit,
  onDelete,
}: {
  user: UserView
  reset?: InviteView
  subusers?: boolean
  onEdit: (u: UserView) => void
  onDelete: (u: UserView) => void
}) {
  const { user: me } = useAuth()
  const toggle = useUpdateUser({
    onSuccess: (u) => toast.success(u.disabled ? `${u.username} disabled` : `${u.username} enabled`),
    onError: failed("change the user"),
  })
  return (
    <TableRow className={u.disabled ? "opacity-60" : undefined}>
      <TableCell className="pl-4">
        <div className="flex items-center gap-3">
          <UserAvatar user={u} />
          <div className="min-w-0">
            <div className="truncate font-medium">
              {userName(u)}
              {u.username === me.username && <span className="ml-2 text-xs text-muted-foreground">(you)</span>}
            </div>
            <div className="truncate font-mono text-xs text-muted-foreground">
              {u.username}
              {u.email && ` · ${u.email}`}
            </div>
            {u.mustChangePassword && (
              <div className="text-xs text-amber-600 dark:text-amber-400">New password pending</div>
            )}
            {reset && (
              <div className="text-xs">
                <InviteState invite={reset} />
              </div>
            )}
            {/* Phones: role and servers here instead of in their own columns. */}
            <div className="truncate text-xs text-muted-foreground sm:hidden">
              {subusers ? `Subuser of ${u.owner}` : roleLabel(u.role)}
              {u.oidc && " · OIDC"} · {plural(u.servers, "server")}
            </div>
          </div>
        </div>
      </TableCell>
      {subusers ? (
        <TableCell className="hidden font-mono text-xs sm:table-cell">{u.owner}</TableCell>
      ) : (
        <>
          <TableCell className="hidden sm:table-cell">
            <div className="flex flex-wrap gap-1">
              <RoleBadge role={u.role} />
              {u.oidc && <OidcBadge />}
            </div>
          </TableCell>
          <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">{u.namespace}</TableCell>
        </>
      )}
      <TableCell className="hidden text-right tabular-nums sm:table-cell">{u.servers}</TableCell>
      <TableCell className="hidden text-right text-xs text-muted-foreground lg:table-cell">
        {u.lastLoginAt ? formatRelativeTime(u.lastLoginAt) : "never"}
      </TableCell>
      <TableCell className="text-center">
        <Switch
          checked={!u.disabled}
          disabled={u.username === me.username || toggle.isPending}
          onCheckedChange={() => toggle.mutate({ name: u.username, changes: { disabled: !u.disabled } })}
          aria-label="Active"
        />
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end">
          {reset ? (
            <InviteActions invite={reset} />
          ) : (
            !u.oidc && u.username !== me.username && <ResetPasswordButton username={u.username} />
          )}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon-sm">
                <MoreHorizontalIcon />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onEdit(u)}>
                <PencilIcon />
                Edit
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" disabled={u.username === me.username} onClick={() => onDelete(u)}>
                <Trash2Icon />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </TableCell>
    </TableRow>
  )
}

/** An invite that was not used yet, in the table of the accounts it creates. */
function InviteRow({ invite: i, subusers }: { invite: InviteView; subusers?: boolean }) {
  return (
    <TableRow>
      <TableCell className="pl-4">
        <div className="flex items-center gap-3">
          <InviteIcon />
          <div className="min-w-0">
            <div className="truncate font-medium">{inviteLabel(i)}</div>
            <div className="truncate text-xs">
              {i.username && i.note && <span className="font-mono text-muted-foreground">{i.username} · </span>}
              <InviteState invite={i} />
            </div>
          </div>
        </div>
      </TableCell>
      {subusers ? (
        <TableCell className="hidden font-mono text-xs sm:table-cell">{i.owner}</TableCell>
      ) : (
        <>
          <TableCell className="hidden sm:table-cell">
            <RoleBadge role={i.role} />
          </TableCell>
          <TableCell className="hidden md:table-cell" />
        </>
      )}
      <TableCell className="hidden text-right tabular-nums sm:table-cell">{subusers ? i.shares.length : "-"}</TableCell>
      <TableCell className="hidden text-right text-xs text-muted-foreground lg:table-cell">-</TableCell>
      <TableCell />
      <TableCell>
        <div className="flex justify-end">
          <InviteActions invite={i} />
        </div>
      </TableCell>
    </TableRow>
  )
}
