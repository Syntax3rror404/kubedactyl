import { Badge } from "@/components/ui/badge"
import { roleLabel } from "@/lib/format"
import type { Role } from "@/lib/types"

export function RoleBadge({ role }: { role: Role }) {
  return <Badge variant={role === "admin" ? "default" : "secondary"}>{roleLabel(role)}</Badge>
}
