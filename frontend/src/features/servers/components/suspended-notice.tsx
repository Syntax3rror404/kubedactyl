import { LockIcon, LockOpenIcon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { suspendFeedback } from "@/features/servers/lib/feedback"
import { useSuspendServer } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Replaces the tabs for the owner of a suspended server. */
export function SuspendedNotice() {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-red-500/30 bg-red-500/5 px-6 py-14 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-red-500/10">
        <LockIcon className="size-6 text-red-500" />
      </div>
      <h2 className="text-lg font-semibold">This server is suspended</h2>
      <p className="max-w-md text-sm text-muted-foreground">
        An administrator has stopped and locked this server. Its files are kept.
      </p>
    </div>
  )
}

/** Tells administrators that the server is suspended, with a shortcut to lift it. */
export function SuspendedBanner({ server }: { server: GameServer }) {
  const suspend = useSuspendServer(server.metadata.name, suspendFeedback)
  return (
    <Callout
      tone="danger"
      icon={<LockIcon />}
      action={
        <Button size="sm" variant="outline" disabled={suspend.isPending} onClick={() => suspend.mutate(false)}>
          {suspend.isPending ? <Spinner /> : <LockOpenIcon />}
          Unsuspend
        </Button>
      }
    >
      <p>
        <span className="font-medium">Suspended.</span> The owner cannot start or use this server until it is
        unsuspended.
      </p>
    </Callout>
  )
}
