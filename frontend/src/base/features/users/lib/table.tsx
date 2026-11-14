// The parts of the user tables (users page, the app's account cards): columns, the role column every app shows and
// how invites are named and split.

import type { ReactNode } from "react"

import { OidcBadge, RoleBadge } from "@/base/components/role-badge"
import { roleLabel } from "@/base/lib/format"
import type { InviteView, UserView } from "@/base/lib/types"

/** A column of a user table between the name and the last login, with the cell of an account and of an invite. */
export interface UserColumn {
  head: string
  /** Classes of the head and the cells: when the column shows ("hidden sm:table-cell") and its alignment. */
  className: string
  /** More classes of the cells (font, color). */
  cellClassName?: string
  user: (u: UserView) => ReactNode
  invite: (i: InviteView) => ReactNode
  /** The value in the line under the name on phones, where the column is hidden. */
  short?: (u: UserView) => string
}

/** One table of the users page: accounts and the invites that create them; a title starts a section of its own. */
export interface UserTableSpec {
  title?: string
  users: UserView[]
  invites: InviteView[]
  columns: UserColumn[]
}

/** The role and the single sign-on badge. */
export const roleColumn: UserColumn = {
  head: "Role",
  className: "hidden sm:table-cell",
  user: (u) => (
    <div className="flex flex-wrap gap-1">
      <RoleBadge role={u.role} />
      {u.oidc && <OidcBadge />}
    </div>
  ),
  invite: (i) => <RoleBadge role={i.role} />,
  short: (u) => roleLabel(u.role) + (u.oidc ? " · OIDC" : ""),
}

/** Who an invite is for: its note, the fixed username or the open choice. */
export const inviteLabel = (i: InviteView) => i.note || i.username || "Username of their choice"

/** Pending invites split into the password reset links (by the username they belong to) and the open invites. */
export function splitInvites(invites: InviteView[]) {
  return {
    resets: new Map(invites.filter((i) => i.reset).map((i) => [i.username, i])),
    open: invites.filter((i) => !i.reset),
  }
}
