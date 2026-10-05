import { useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"

/** Asks for a single name (new folder, new file, rename). */
export function FileNameDialog({
  open,
  title,
  label,
  initial = "",
  pending,
  onClose,
  onSubmit,
}: {
  open: boolean
  title: string
  label: string
  initial?: string
  pending?: boolean
  onClose: () => void
  onSubmit: (name: string) => void
}) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      {/* The next dialog (e.g. the editor after "New file") may open right away; returning the focus
          to the trigger would count as a click outside it and close it again. */}
      <DialogContent className="sm:max-w-md" onCloseAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{label}</DialogDescription>
        </DialogHeader>
        {/* mounted fresh on every open, so the input starts with `initial` */}
        <NameForm initial={initial} pending={pending} onClose={onClose} onSubmit={onSubmit} />
      </DialogContent>
    </Dialog>
  )
}

function NameForm({
  initial,
  pending,
  onClose,
  onSubmit,
}: {
  initial: string
  pending?: boolean
  onClose: () => void
  onSubmit: (name: string) => void
}) {
  const [value, setValue] = useState(initial)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (value.trim()) onSubmit(value.trim())
  }
  return (
    <form onSubmit={submit} className="contents">
      <Input autoFocus value={value} onChange={(e) => setValue(e.target.value)} className="font-mono" />
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={!value.trim() || pending}>
          {pending && <Spinner />}
          Save
        </Button>
      </DialogFooter>
    </form>
  )
}
