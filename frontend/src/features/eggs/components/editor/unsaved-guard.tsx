import { useBlocker } from "react-router"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { useBeforeUnload } from "@/hooks/use-before-unload"

/** Asks before leaving the page (in the app or by closing the tab) while there are unsaved changes. */
export function UnsavedGuard({ when }: { when: boolean }) {
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) => when && currentLocation.pathname !== nextLocation.pathname,
  )
  useBeforeUnload(when)
  return (
    <ConfirmDialog
      open={blocker.state === "blocked"}
      onOpenChange={(open) => !open && blocker.reset?.()}
      title="Discard your changes?"
      description="The egg has unsaved changes that are lost when you leave this page."
      confirmLabel="Discard"
      cancelLabel="Keep editing"
      destructive
      onConfirm={() => blocker.proceed?.()}
    />
  )
}
