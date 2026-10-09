import { ForkliftIcon, LockIcon, LockOpenIcon, Trash2Icon, XIcon } from "lucide-react"
import { toast } from "sonner"

import { Callout } from "@/components/common/callout"
import { ProgressBar } from "@/components/common/progress-bar"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { suspendFeedback } from "@/features/servers/lib/feedback"
import { failed } from "@/lib/notify"
import { useCancelMigration, useSuspendServer } from "@/lib/queries"
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

/**
 * Replaces the tabs while the server files move to another storage class: the step, the files copied and, for
 * admins, a button that calls the migration off until the server switches to the new volume.
 */
export function MigrationNotice({ server, isAdmin }: { server: GameServer; isAdmin: boolean }) {
  const to = server.spec.storageClass
  const m = server.status?.migration?.to === to ? server.status?.migration : undefined
  const step = m?.step ?? "Preparing"
  const cancel = useCancelMigration(server.metadata.name, {
    onSuccess: () => toast.success("Storage migration cancelled"),
    onError: failed("cancel the storage migration"),
  })
  const text = {
    Preparing: `Stopping the server and creating a volume of ${to}.`,
    Copying: `Copying the files to ${to}: ${(m?.done ?? 0).toLocaleString("en-US")} of ${(m?.total ?? 0).toLocaleString("en-US")} files.`,
    Switching: "Files copied and checked, switching to the new volume.",
    Failed: "",
  }[step]
  return (
    <Callout
      tone="info"
      icon={<ForkliftIcon />}
      title="Server blocked, storage migration in progress"
      progress
      action={
        isAdmin &&
        step !== "Switching" && (
          <Button size="sm" variant="outline" disabled={cancel.isPending} onClick={() => cancel.mutate()}>
            {cancel.isPending ? <Spinner /> : <XIcon />}
            Cancel
          </Button>
        )
      }
    >
      <p className="text-muted-foreground">
        {text} {server.status?.message}
      </p>
      <ProgressBar
        value={step === "Switching" ? 1 : step === "Copying" && m?.total ? m.done : undefined}
        max={step === "Switching" ? 1 : m?.total}
        className="mt-2 h-1.5"
      />
    </Callout>
  )
}
