import { MoreHorizontalIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { RoleBadge } from "@/components/common/role-badge"
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
import { useAuth } from "@/hooks/use-auth"
import { formatRelativeTime, plural, roleLabel, userName } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useUpdateUser } from "@/lib/queries"
import type { UserView } from "@/lib/types"

/** Users with role, state, servers and actions. */
export function UserTable({
  users,
  onEdit,
  onDelete,
}: {
  users: UserView[]
  onEdit: (u: UserView) => void
  onDelete: (u: UserView) => void
}) {
  const { user: me } = useAuth()
  const toggle = useUpdateUser({
    onSuccess: (u) => toast.success(u.disabled ? `${u.username} disabled` : `${u.username} enabled`),
    onError: failed("change the user"),
  })
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="pl-4">User</TableHead>
          <TableHead className="hidden sm:table-cell">Role</TableHead>
          <TableHead className="hidden md:table-cell">K8s Namespace</TableHead>
          <TableHead className="hidden text-right sm:table-cell">Servers</TableHead>
          <TableHead className="hidden text-right lg:table-cell">Last login</TableHead>
          <TableHead className="text-center">Active</TableHead>
          <TableHead className="w-12" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((u) => (
          <TableRow key={u.username} className={u.disabled ? "opacity-60" : undefined}>
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
                  {/* Phones: role and servers here instead of in their own columns. */}
                  <div className="truncate text-xs text-muted-foreground sm:hidden">
                    {roleLabel(u.role)} · {plural(u.servers, "server")}
                  </div>
                </div>
              </div>
            </TableCell>
            <TableCell className="hidden sm:table-cell">
              <RoleBadge role={u.role} />
            </TableCell>
            <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
              {u.namespace}
            </TableCell>
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
                  <DropdownMenuItem
                    variant="destructive"
                    disabled={u.username === me.username}
                    onClick={() => onDelete(u)}
                  >
                    <Trash2Icon />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
