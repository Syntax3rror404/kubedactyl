import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

/**
 * Dialog of an egg feature: closing it dismisses the matched console line. The dialog stays open while the confirm
 * request runs; `action` replaces the button (a link, for example). Without both it only explains (Close).
 */
export function FeaturePrompt({
  open,
  onClose,
  title,
  description,
  children,
  confirm,
  action,
}: {
  open: boolean
  onClose: () => void
  title: string
  description: React.ReactNode
  /** Fields between the description and the buttons. */
  children?: React.ReactNode
  /** Omitted when the user cannot fix it. */
  confirm?: { label: string; pending: boolean; onConfirm: () => void }
  action?: React.ReactNode
}) {
  return (
    <AlertDialog open={open} onOpenChange={(o) => !o && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription render={<div className="space-y-2" />}>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        {children}
        <AlertDialogFooter>
          <AlertDialogCancel>{confirm || action ? "Cancel" : "Close"}</AlertDialogCancel>
          {confirm && (
            <AlertDialogAction disabled={confirm.pending} onClick={confirm.onConfirm}>
              {confirm.label}
            </AlertDialogAction>
          )}
          {action}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
