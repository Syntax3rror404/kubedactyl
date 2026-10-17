import { Link } from "react-router"

import { AccessSelect, type AccessChoice } from "@/components/common/access-select"
import { UserAvatar } from "@/components/common/user-avatar"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { FieldError } from "@/components/ui/field"
import type { SettingsCardProps } from "@/features/servers/lib/settings-draft"
import { useAuth } from "@/hooks/use-auth"
import { serverOwner, userName } from "@/lib/format"
import { useUsers } from "@/lib/queries"

/** Settings page: which subusers of the owner see the server (view) or also start, stop and command it (control). */
export function SharingCard({ server, draft, onChange, errors }: SettingsCardProps) {
  const { user: me } = useAuth()
  const owner = serverOwner(server)
  const subusers = useUsers().data?.filter((u) => u.owner && u.owner === owner) ?? []
  const accessOf = (user: string) => draft.sharedWith.find((s) => s.user === user)?.access ?? "none"
  const set = (user: string, access: AccessChoice) => {
    const rest = draft.sharedWith.filter((s) => s.user !== user)
    onChange({ sharedWith: access === "none" ? rest : [...rest, { user, access }] })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Subusers</CardTitle>
        <CardDescription>Files, backups and settings stay with the owner.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {subusers.length === 0 && owner !== me.username ? (
          <p className="text-sm text-muted-foreground">The owner has no subusers.</p>
        ) : subusers.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No subusers yet. Invite them on the{" "}
            <Link to="/account" className="underline underline-offset-4 hover:text-foreground">
              account page
            </Link>
            .
          </p>
        ) : (
          subusers.map((u) => (
            <div key={u.username} className="flex items-center gap-3">
              <UserAvatar user={u} />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium">{userName(u)}</div>
                <div className="truncate font-mono text-xs text-muted-foreground">{u.username}</div>
              </div>
              <AccessSelect
                value={accessOf(u.username)}
                onChange={(v) => set(u.username, v)}
                label={`Access of ${u.username}`}
              />
            </div>
          ))
        )}
        <FieldError>{errors.sharedWith}</FieldError>
      </CardContent>
    </Card>
  )
}
