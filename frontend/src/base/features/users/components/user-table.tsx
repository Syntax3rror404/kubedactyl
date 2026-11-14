import { MoreHorizontalIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { UserAvatar } from "@/base/components/user-avatar"
import {
  InviteActions,
  InviteIcon,
  InviteState,
  PasswordResetAction,
} from "@/base/features/users/components/invite-actions"
import { inviteLabel, splitInvites, type UserColumn } from "@/base/features/users/lib/table"
import { useAuth } from "@/base/hooks/use-auth"
import { formatRelativeTime, userName } from "@/base/lib/format"
import { failed, useUpdateUser } from "@/base/lib/queries"
import type { InviteView, UserView } from "@/base/lib/types"
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
import { cn } from "@/lib/utils"

// Phones: the name cell takes the width left and wraps its lines, so the actions stay on screen.
const nameCell = "w-full max-w-0 pl-4 whitespace-normal sm:w-auto sm:max-w-none sm:whitespace-nowrap"

/**
 * Users with the app's columns (`columns`), last login, state and actions, followed by the invites that were not used
 * yet. Local accounts get a password reset link; a pending one is shown in the row.
 */
export function UserTable({
  users,
  invites,
  columns,
  onEdit,
  onDelete,
}: {
  users: UserView[]
  invites: InviteView[]
  columns: UserColumn[]
  onEdit: (u: UserView) => void
  onDelete: (u: UserView) => void
}) {
  const { resets, open } = splitInvites(invites)
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="pl-4">User</TableHead>
          {columns.map((c) => (
            <TableHead key={c.head} className={c.className}>
              {c.head}
            </TableHead>
          ))}
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
            columns={columns}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
        {open.map((i) => (
          <InviteRow key={i.id} invite={i} columns={columns} />
        ))}
      </TableBody>
    </Table>
  )
}

function UserRow({
  user: u,
  reset,
  columns,
  onEdit,
  onDelete,
}: {
  user: UserView
  reset?: InviteView
  columns: UserColumn[]
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
      <TableCell className={nameCell}>
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
            {u.mustChangePassword && <div className="text-xs text-warning">New password pending</div>}
            {reset && (
              <div className="text-xs">
                <InviteState invite={reset} />
              </div>
            )}
            {/* Phones: the columns here instead of in their own cells. */}
            <div className="text-xs text-muted-foreground sm:hidden">
              {columns.flatMap((c) => c.short?.(u) ?? []).join(" · ")}
            </div>
          </div>
        </div>
      </TableCell>
      {columns.map((c) => (
        <TableCell key={c.head} className={cn(c.className, c.cellClassName)}>
          {c.user(u)}
        </TableCell>
      ))}
      <TableCell className="hidden text-right text-xs text-muted-foreground lg:table-cell">
        {u.lastLoginAt ? formatRelativeTime(u.lastLoginAt) : "never"}
      </TableCell>
      <TableCell className="text-center">
        <Switch
          checked={!u.disabled}
          disabled={u.username === me.username || toggle.isPending}
          onCheckedChange={() => toggle.mutate({ name: u.username, changes: { disabled: !u.disabled } })}
          aria-label={`${u.username} active`}
        />
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end">
          <PasswordResetAction username={u.username} reset={reset} canReset={!u.oidc && u.username !== me.username} />
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button variant="ghost" size="icon-sm" aria-label={`Actions for ${u.username}`} />}
            >
              <MoreHorizontalIcon />
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
function InviteRow({ invite: i, columns }: { invite: InviteView; columns: UserColumn[] }) {
  return (
    <TableRow>
      <TableCell className={nameCell}>
        <div className="flex items-center gap-3">
          <InviteIcon />
          <div className="min-w-0">
            <div className="truncate font-medium">{inviteLabel(i)}</div>
            <div className="text-xs">
              {i.username && i.note && <span className="font-mono text-muted-foreground">{i.username} · </span>}
              <InviteState invite={i} />
            </div>
          </div>
        </div>
      </TableCell>
      {columns.map((c) => (
        <TableCell key={c.head} className={cn(c.className, c.cellClassName)}>
          {c.invite(i)}
        </TableCell>
      ))}
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
