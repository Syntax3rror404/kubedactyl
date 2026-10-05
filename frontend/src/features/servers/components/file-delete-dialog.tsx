import { ConfirmDialog } from "@/components/common/confirm-dialog"

/** Confirmation before deleting files. */
export function FileDeleteDialog({
  names,
  onClose,
  onConfirm,
}: {
  names: string[] | null
  onClose: () => void
  onConfirm: (names: string[]) => void
}) {
  const title = names?.length === 1 ? names[0] : `${names?.length ?? 0} entries`
  return (
    <ConfirmDialog
      open={!!names}
      onOpenChange={(o) => !o && onClose()}
      title={`Delete ${title}?`}
      description="Folders are deleted with all their contents. This cannot be undone."
      confirmLabel="Delete"
      destructive
      onConfirm={() => names && onConfirm(names)}
    />
  )
}
