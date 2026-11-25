import { useMemo, useState, type ReactNode } from "react"
import { DownloadIcon } from "lucide-react"

import { roles } from "@/app/config"
import { Callout } from "@/base/components/callout"
import { CopyField } from "@/base/components/copy-button"
import { UsernameField, UsernameHint } from "@/base/components/username-field"
import { RoleSelect } from "@/base/features/users/components/user-dialog"
import { downloadQrCode, qrModules } from "@/base/features/users/lib/qr-code"
import { useDraft } from "@/base/hooks/use-draft"
import { formatDate } from "@/base/lib/format"
import { failed, useCreateInvite } from "@/base/lib/queries"
import type { CreatedInvite, CreateInviteRequest } from "@/base/lib/types"
import { fieldErrors } from "@/base/lib/validation"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { cn } from "@/lib/utils"

/** Longest invite note (the panel checks the same limit). */
const MAX_NOTE = 200

/** What an app adds to an invite in place of the role (a subuser of the caller and the servers it sees). */
export interface InviteExtra<T> {
  title: string
  description: string
  initial: T
  /** Fields below the note. */
  fields: (value: T, onChange: (value: T) => void, errors: Record<string, string>) => ReactNode
  /** The app's parts of the request. */
  request: (value: T) => Partial<CreateInviteRequest>
}

/**
 * Creates an invite link and then shows it once: as QR code and as text to send in a messenger. With extra the app
 * decides what the account becomes instead of the role picked here.
 */
export function InviteDialog<T>({
  open,
  onClose,
  extra,
}: {
  open: boolean
  onClose: () => void
  extra?: InviteExtra<T>
}) {
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const close = () => {
    setCreated(null)
    onClose()
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && close()}>
      <DialogContent className="sm:max-w-md">
        {open &&
          (created ? (
            <InviteLink invite={created} onClose={close} />
          ) : (
            <InviteForm onCreated={setCreated} extra={extra} />
          ))}
      </DialogContent>
    </Dialog>
  )
}

function InviteForm<T>({ onCreated, extra }: { onCreated: (invite: CreatedInvite) => void; extra?: InviteExtra<T> }) {
  const { draft, set } = useDraft({ username: "", role: roles[0].value, note: "" })
  const [extraValue, setExtraValue] = useState(extra?.initial)
  // The app's value exists exactly when the app adds to the invite.
  const app = extra && extraValue !== undefined ? { ...extra, value: extraValue } : undefined
  const create = useCreateInvite({ onSuccess: onCreated, onError: failed("create the invite") })
  const errors = fieldErrors(create.error)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    create.mutate({
      username: draft.username || undefined,
      note: draft.note || undefined,
      ...(app ? app.request(app.value) : { role: draft.role }),
    })
  }

  return (
    <form onSubmit={submit}>
      <DialogHeader>
        <DialogTitle>{extra?.title ?? "Invite"}</DialogTitle>
        <DialogDescription>
          {extra?.description ?? "One person can create an account with it. It works once, for 7 days."}
        </DialogDescription>
      </DialogHeader>
      <FieldGroup className="py-4">
        <div className={extra ? "grid gap-4" : "grid gap-4 sm:grid-cols-2"}>
          <UsernameField
            id="i-name"
            value={draft.username}
            onChange={(v) => set("username", v)}
            error={errors.username}
            placeholder="their choice"
            hint={null}
          />
          {!extra && (
            <Field>
              <FieldLabel htmlFor="i-role">Role</FieldLabel>
              <RoleSelect id="i-role" value={draft.role} onChange={(v) => set("role", v)} />
            </Field>
          )}
        </div>
        <UsernameHint error={errors.username} hint="Leave the username empty to let the invited person choose it." />
        <Field>
          <FieldLabel htmlFor="i-note">Note</FieldLabel>
          <Input
            id="i-note"
            value={draft.note}
            onChange={(e) => set("note", e.target.value)}
            placeholder="Who the invite is for"
            maxLength={MAX_NOTE}
          />
        </Field>
        {app?.fields(app.value, setExtraValue, errors)}
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

/** The link of a new or renewed invite or reset link, shown once: QR code, text to copy and a PNG download. */
function InviteLink({ invite, onClose }: { invite: CreatedInvite; onClose: () => void }) {
  const link = `${window.location.origin}/invite?token=${encodeURIComponent(invite.token)}`
  return (
    <>
      <DialogHeader>
        <DialogTitle>{invite.reset ? `Password reset link for ${invite.username}` : "Invite link"}</DialogTitle>
        <DialogDescription>
          Let the person scan the code or send them the link. It works once, until {formatDate(invite.expiresAt)}.
        </DialogDescription>
      </DialogHeader>
      <div className="flex flex-col items-center gap-4 py-4">
        <QrCode value={link} className="size-56" />
        <CopyField value={link} label="Copy link" wrap className="w-full" />
        <Callout tone="warning" className="w-full">
          Copy or save it now, it is not shown again. Renewing it creates a new link.
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

/** A dialog that shows the link of a renewed invite or a new reset link while there is one. */
export function InviteLinkDialog({ invite, onClose }: { invite: CreatedInvite | null; onClose: () => void }) {
  return (
    <Dialog open={!!invite} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        {invite && <InviteLink invite={invite} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  )
}

/** QR code as SVG, black on white in both themes, so every camera reads it. */
function QrCode({ value, className }: { value: string; className?: string }) {
  const data = useMemo(() => qrModules(value), [value])
  const path = data.flatMap((row, y) => row.map((dark, x) => (dark ? `M${x} ${y}h1v1h-1z` : ""))).join("")
  return (
    <svg
      viewBox={`0 0 ${data.length} ${data.length}`}
      shapeRendering="crispEdges"
      role="img"
      aria-label="QR code"
      className={cn("rounded-lg bg-white", className)}
    >
      <path d={path} fill="#000" />
    </svg>
  )
}
