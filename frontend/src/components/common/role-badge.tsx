import { Badge } from "@/components/ui/badge"
import { roleLabel } from "@/lib/format"
import type { Role } from "@/lib/types"

/** The role of an account, administrators highlighted. */
export function RoleBadge({ role }: { role: Role }) {
  return <Badge variant={role === "admin" ? "default" : "secondary"}>{roleLabel(role)}</Badge>
}

/** Marks an account linked to the identity provider (single sign-on), which sets its profile and role. */
export function OidcBadge() {
  return (
    <Badge variant="outline" title="Linked to the identity provider">
      OIDC
    </Badge>
  )
}
