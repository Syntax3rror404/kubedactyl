import type { InviteView } from "@/lib/types"

/** Who an invite is for: its note, the fixed username or the open choice. */
export const inviteLabel = (i: InviteView) => i.note || i.username || "Username of their choice"

/** Pending invites split into the password reset links (by the username they belong to) and the open invites. */
export function splitInvites(invites: InviteView[]) {
  return {
    resets: new Map(invites.filter((i) => i.reset).map((i) => [i.username, i])),
    open: invites.filter((i) => !i.reset),
  }
}
