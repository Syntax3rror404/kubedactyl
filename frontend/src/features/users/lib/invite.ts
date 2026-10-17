import type { InviteView } from "@/lib/types"

/** Who an invite is for: its note, the fixed username or the open choice. */
export const inviteLabel = (i: InviteView) => i.note || i.username || "Username of their choice"
