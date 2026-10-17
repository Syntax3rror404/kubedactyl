import { useState } from "react"
import { DownloadIcon } from "lucide-react"

import { AccessSelect, type AccessChoice } from "@/components/common/access-select"
import { Callout } from "@/components/common/callout"
import { CopyField } from "@/components/common/copy-button"
import { UsernameField, UsernameHint } from "@/components/common/username-field"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldError, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { QrCode } from "@/features/users/components/qr-code"
import { RoleSelect } from "@/features/users/components/user-dialog"
import { downloadQrCode } from "@/features/users/lib/qr-code"
import { useAuth } from "@/hooks/use-auth"
import { useDraft } from "@/hooks/use-draft"
import { formatDate, serverName, serverOwner } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useCreateInvite, useServers } from "@/lib/queries"
import type { CreatedInvite, Role } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/**
 * Creates an invite link and then shows it once: as QR code and as text to send in a messenger. With subuser the
 * account becomes a subuser of the signed-in user and sees the servers picked here (server is picked at the start).
 */
export function InviteDialog({
  open,
  onClose,
  subuser,
  server,
}: {
  open: boolean
  onClose: () => void
  subuser?: boolean
  server?: string
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
            <InviteForm onCreated={setCreated} subuser={subuser} server={server} />
          ))}
      </DialogContent>
    </Dialog>
  )
}

function InviteForm({
  onCreated,
  subuser,
  server,
}: {
  onCreated: (invite: CreatedInvite) => void
  subuser?: boolean
  server?: string
}) {
  const { draft, set } = useDraft({
    username: "",
    role: "user" as Role,
    note: "",
    shares: (server ? { [server]: "view" } : {}) as Record<string, AccessChoice>,
  })
  const create = useCreateInvite({ onSuccess: onCreated, onError: failed("create the invite") })
  const errors = fieldErrors(create.error)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    create.mutate({
      username: draft.username || undefined,
      role: subuser ? undefined : draft.role,
      note: draft.note || undefined,
      subuser,
      shares: Object.entries(draft.shares).flatMap(([s, access]) => (access === "none" ? [] : [{ server: s, access }])),
    })
  }

  return (
    <form onSubmit={submit}>
      <DialogHeader>
        <DialogTitle>{subuser ? "Invite a subuser" : "Invite"}</DialogTitle>
        <DialogDescription>
          {subuser
            ? "They see only the servers you share with them. The link works once, for 7 days."
            : "One person can create an account with it. It works once, for 7 days."}
        </DialogDescription>
      </DialogHeader>
      <FieldGroup className="py-4">
        <div className={subuser ? "grid gap-4" : "grid gap-4 sm:grid-cols-2"}>
          <UsernameField
            id="i-name"
            value={draft.username}
            onChange={(v) => set("username", v)}
            error={errors.username}
            placeholder="their choice"
            hint={null}
          />
          {!subuser && (
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
          />
        </Field>
        {subuser && (
          <ShareFields
            value={draft.shares}
            onChange={(shares) => set("shares", shares)}
            error={Object.entries(errors).find(([k]) => k.startsWith("shares"))?.[1]}
          />
        )}
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

/** The own servers with the access the subuser gets to each. */
function ShareFields({
  value,
  onChange,
  error,
}: {
  value: Record<string, AccessChoice>
  onChange: (value: Record<string, AccessChoice>) => void
  error?: string
}) {
  const { user } = useAuth()
  const own = useServers().data?.filter((s) => serverOwner(s) === user.username)
  if (!own?.length) return null
  return (
    <FieldSet>
      <FieldLegend variant="label">Servers</FieldLegend>
      <div className="space-y-2">
        {own.map((s) => (
          <div key={s.metadata.name} className="flex items-center gap-3">
            <span className="min-w-0 flex-1 truncate text-sm">{serverName(s)}</span>
            <AccessSelect
              value={value[s.metadata.name] ?? "none"}
              onChange={(access) => onChange({ ...value, [s.metadata.name]: access })}
              label={`Access to ${serverName(s)}`}
            />
          </div>
        ))}
      </div>
      <FieldError>{error}</FieldError>
    </FieldSet>
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
