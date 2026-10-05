import { useState } from "react"
import { DownloadIcon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { CopyButton } from "@/components/common/copy-button"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { QrCode } from "@/features/users/components/qr-code"
import { downloadQrCode } from "@/features/users/lib/qr-code"
import { useDraft } from "@/hooks/use-draft"
import { formatDate } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useCreateInvite } from "@/lib/queries"
import type { CreatedInvite, Role } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/** Creates an invite link and then shows it once: as QR code and as text to send in a messenger. */
export function InviteDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const close = () => {
    setCreated(null)
    onClose()
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && close()}>
      <DialogContent className="sm:max-w-md">
        {open && (created ? <InviteLink invite={created} onClose={close} /> : <InviteForm onCreated={setCreated} />)}
      </DialogContent>
    </Dialog>
  )
}

function InviteForm({ onCreated }: { onCreated: (invite: CreatedInvite) => void }) {
  const { draft, set } = useDraft({ username: "", role: "user" as Role, note: "" })
  const create = useCreateInvite({ onSuccess: onCreated, onError: failed("create the invite") })
  const errors = fieldErrors(create.error)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    create.mutate({ username: draft.username || undefined, role: draft.role, note: draft.note || undefined })
  }

  return (
    <form onSubmit={submit}>
      <DialogHeader>
        <DialogTitle>Invite</DialogTitle>
        <DialogDescription>One person can create an account with it. It works once, for 7 days.</DialogDescription>
      </DialogHeader>
      <FieldGroup className="py-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field data-invalid={!!errors.username}>
            <FieldLabel htmlFor="i-name">Username</FieldLabel>
            <Input
              id="i-name"
              value={draft.username}
              onChange={(e) => set("username", e.target.value.toLowerCase())}
              aria-invalid={!!errors.username}
              placeholder="their choice"
              className="font-mono"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="i-role">Role</FieldLabel>
            <Select value={draft.role} onValueChange={(v) => set("role", v as Role)}>
              <SelectTrigger id="i-role" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="admin">Administrator</SelectItem>
              </SelectContent>
            </Select>
          </Field>
        </div>
        {errors.username ? (
          <FieldError>{errors.username}</FieldError>
        ) : (
          <FieldDescription>Leave the username empty to let the invited person choose it.</FieldDescription>
        )}
        <Field>
          <FieldLabel htmlFor="i-note">Note</FieldLabel>
          <Input
            id="i-note"
            value={draft.note}
            onChange={(e) => set("note", e.target.value)}
            placeholder="Who the invite is for"
          />
        </Field>
      </FieldGroup>
      <DialogFooter>
        <Button type="submit" disabled={create.isPending}>
          {create.isPending && <Spinner />}
          Create link
        </Button>
      </DialogFooter>
    </form>
  )
}

/** The link of a new or renewed invite, shown once: QR code, text to copy and a PNG download. */
export function InviteLink({ invite, onClose }: { invite: CreatedInvite; onClose: () => void }) {
  const link = `${window.location.origin}/invite?token=${encodeURIComponent(invite.token)}`
  return (
    <>
      <DialogHeader>
        <DialogTitle>Invite link</DialogTitle>
        <DialogDescription>
          Let the person scan the code or send them the link. It works once, until {formatDate(invite.expiresAt)}.
        </DialogDescription>
      </DialogHeader>
      <div className="flex flex-col items-center gap-4 py-4">
        <QrCode value={link} className="size-56" />
        <div className="flex w-full items-center gap-2 rounded-lg bg-muted px-3 py-1.5">
          <code className="flex-1 font-mono text-xs break-all">{link}</code>
          <CopyButton value={link} label="Copy link" />
        </div>
        <Callout tone="warning" className="w-full">
          Copy or save it now, it is not shown again. Renewing the invite creates a new link.
        </Callout>
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={() => downloadQrCode(link, `invite-${invite.username || invite.id}.png`)}>
          <DownloadIcon />
          Save QR code
        </Button>
        <Button onClick={onClose}>Done</Button>
      </DialogFooter>
    </>
  )
}
