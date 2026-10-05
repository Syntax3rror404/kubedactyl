import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useConsoleMatch, type Subscribe } from "@/features/servers/egg-features/use-console-match"

// Same patterns as Pterodactyl and Pelican (case-insensitive substrings).
const pidPatterns = [
  "pthread_create failed",
  "failed to create thread",
  "unable to create thread",
  "unable to create native thread",
  "unable to create new native thread",
  'exception in thread "craft async scheduler management thread"',
]
const diskPatterns = ["steamcmd needs 250mb of free disk space to update", "0x202 after update job"]

function Notice({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean
  onClose: () => void
  title: string
  children: React.ReactNode
}) {
  return (
    <AlertDialog open={open} onOpenChange={(o) => !o && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2">{children}</div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogAction>Close</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

/** Egg feature "pid_limit": the server could not create threads (memory or process limit). */
export function PidLimitNotice({
  subscribe,
  running,
  isAdmin,
}: {
  subscribe: Subscribe
  running: boolean
  isAdmin: boolean
}) {
  const { line, dismiss } = useConsoleMatch(subscribe, pidPatterns)
  return (
    <Notice
      open={!!line && !running}
      onClose={dismiss}
      title={isAdmin ? "Memory or process limit reached" : "Possible resource limit reached"}
    >
      {isAdmin ? (
        <>
          <p>The server hit its memory or process limit.</p>
          <p>
            Give it more memory, or raise <code>podPidsLimit</code> of the node's kubelet.
          </p>
        </>
      ) : (
        <p>The server needs more resources. Send an administrator this error: “pthread_create failed”.</p>
      )}
    </Notice>
  )
}

/** Egg feature "steam_disk_space": SteamCMD ran out of disk space. */
export function SteamDiskSpaceNotice({
  subscribe,
  running,
  isAdmin,
}: {
  subscribe: Subscribe
  running: boolean
  isAdmin: boolean
}) {
  const { line, dismiss } = useConsoleMatch(subscribe, diskPatterns)
  return (
    <Notice open={!!line && !running} onClose={dismiss} title="Out of disk space">
      <p>The server ran out of disk space and cannot finish the install or update.</p>
      <p>
        {isAdmin ? "Increase the disk size or delete files." : "Delete files or ask an administrator for more space."}
      </p>
    </Notice>
  )
}
