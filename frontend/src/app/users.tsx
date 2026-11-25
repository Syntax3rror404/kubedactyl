// What this app adds to the shared users and account pages (base/features/users, base/features/account): the
// columns of the user tables (namespace, servers) with the subusers in a table of their own, what deleting and
// creating an account means here, and the subusers card of the account page.

import { roleColumn, type UserColumn, type UserTableSpec } from "@/base/features/users/table"
import { plural } from "@/base/lib/format"
import type { InviteView, UserView } from "@/base/lib/types"
import { SubusersCard } from "@/features/account/subusers-card"

const namespaceColumn: UserColumn = {
  head: "K8s Namespace",
  className: "hidden md:table-cell",
  cellClassName: "text-muted-foreground",
  user: (u) => u.namespace,
  invite: () => null,
}

const serversColumn = (head: string, invite: (i: InviteView) => string | number): UserColumn => ({
  head,
  className: "hidden text-right sm:table-cell",
  cellClassName: "tabular-nums",
  user: (u) => u.servers,
  invite,
  short: (u) => plural(u.servers, "server"),
})

const ownerColumn: UserColumn = {
  head: "Owner",
  className: "hidden sm:table-cell",
  user: (u) => u.owner,
  invite: (i) => i.owner,
  short: (u) => `Subuser of ${u.owner}`,
}

/**
 * The tables of the users page: the accounts, then the subusers of all users (while there are some). Password reset
 * links belong to an account of either table; the rows find them by username.
 */
export function userTables(users: UserView[], invites: InviteView[]): UserTableSpec[] {
  const subusers = users.filter((u) => u.owner)
  const subuserInvites = invites.filter((i) => i.reset || i.owner)
  const tables: UserTableSpec[] = [
    {
      users: users.filter((u) => !u.owner),
      invites: invites.filter((i) => i.reset || !i.owner),
      columns: [roleColumn, namespaceColumn, serversColumn("Servers", () => "-")],
    },
  ]
  if (subusers.length > 0 || subuserInvites.some((i) => !i.reset))
    tables.push({
      title: "Subusers",
      users: subusers,
      invites: subuserInvites,
      columns: [ownerColumn, serversColumn("Shared servers", (i) => i.shares.length)],
    })
  return tables
}

/** Under the title of the users page. */
export const usersHint = undefined

/** What deleting an account removes: the text of the confirmation and the note of the message afterwards. */
export function deleteUserNote(u: UserView) {
  if (u.owner) return { confirm: "The account is deleted permanently and loses access to the servers shared with it." }
  return {
    confirm: (
      <>
        All {u.servers} game server(s) of this user, their data, the namespace{" "}
        <span className="font-medium">{u.namespace}</span> and their subusers are deleted permanently.
      </>
    ),
    done: "Servers, data and the namespace are being removed.",
  }
}

/** The hint of the new user dialog, and the note of the message once the account exists. */
export const newUserHint = "The user gets an own namespace for game servers."
export const createdUserNote = (u: UserView) => `Namespace ${u.namespace}`

/** A role the user dialog shows read only, as text: a subuser has its owner instead. */
export const fixedRole = (u: UserView) => (u.owner ? `Subuser of ${u.owner}` : undefined)

/** Next to the role on the account page: the namespace, or the owner of a subuser. */
export const accountDetail = (u: UserView) =>
  u.owner ? <span className="text-xs">Subuser of {u.owner}</span> : <span>{u.namespace}</span>

/** Cards of the account page below the API tokens. */
export const accountCards = [SubusersCard]
