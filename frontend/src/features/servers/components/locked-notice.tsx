import { LockIcon, LockOpenIcon, Trash2Icon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { suspendFeedback } from "@/features/servers/lib/feedback"
import { useSuspendServer } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Replaces the tabs for the owner of a suspended server. */
export function SuspendedNotice() {
  return (
    <LockedNotice icon={<LockIcon />} title="This server is suspended">
      An administrator has stopped and locked this server. Its files are kept.
    </LockedNotice>
  )
}

/** Replaces the tabs while a deleted server waits for its volume to be released. */
export function RemovingNotice() {
  return (
    <LockedNotice icon={<Trash2Icon />} title="Removal scheduled">
      The server disappears as soon as its volume is released.
    </LockedNotice>
  )
}

function LockedNotice({ icon, title, children }: { icon: React.ReactNode; title: string; children: string }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-red-500/30 bg-red-500/5 px-6 py-14 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-red-500/10 [&_svg]:size-6 [&_svg]:text-red-500">
        {icon}
      </div>
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="max-w-md text-sm text-muted-foreground">{children}</p>
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
