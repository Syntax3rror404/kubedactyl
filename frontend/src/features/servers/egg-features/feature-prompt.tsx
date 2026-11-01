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
 * Dialog of an egg feature: closing it dismisses the matched console line. The confirm button keeps the dialog open
 * while its request runs; `action` replaces it (a link, for example). Without both it only explains (Close).
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
          <AlertDialogDescription asChild>
            <div className="space-y-2">{description}</div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        {children}
        <AlertDialogFooter>
          <AlertDialogCancel>{confirm || action ? "Cancel" : "Close"}</AlertDialogCancel>
          {confirm && (
            <AlertDialogAction
              disabled={confirm.pending}
              onClick={(e) => {
                e.preventDefault()
                confirm.onConfirm()
              }}
            >
              {confirm.label}
            </AlertDialogAction>
          )}
          {action}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
