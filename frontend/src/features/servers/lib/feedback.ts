import { toast } from "sonner"

import { failed } from "@/base/lib/queries"
import type { MutationCallbacks } from "@/base/lib/queries"
import type { GameServer, PowerSignal } from "@/lib/types"

/** What the user sees after a power action, which several components offer. */
export const powerFeedback: MutationCallbacks<void, PowerSignal> = {
  onError: failed("run the power action"),
}

/** What the user sees after suspending or unsuspending, which the danger zone and the banner offer. */
export const suspendFeedback: MutationCallbacks<GameServer, boolean> = {
  onSuccess: (gs) => toast.success(gs.spec.suspended ? "Server suspended" : "Server unsuspended"),
  onError: failed("change the suspension"),
}
