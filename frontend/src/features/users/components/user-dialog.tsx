import { toast } from "sonner"

import { CopyButton } from "@/components/common/copy-button"
import { UnveilPassword } from "@/components/common/unveil-password"
import { UsernameField } from "@/components/common/username-field"
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
import { Switch } from "@/components/ui/switch"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useCreateUser, useUpdateUser } from "@/lib/queries"
import type { Role, UserView } from "@/lib/types"
import { fieldErrors, MIN_PASSWORD_LENGTH } from "@/lib/validation"

/** Create a user (user = null) or edit an existing one. */
export function UserDialog({ open, user, onClose }: { open: boolean; user: UserView | null; onClose: () => void }) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">{open && <UserForm user={user} onClose={onClose} />}</DialogContent>
    </Dialog>
  )
}

function UserForm({ user, onClose }: { user: UserView | null; onClose: () => void }) {
  const { draft, set } = useDraft({
    username: user?.username ?? "",
    displayName: user?.displayName ?? "",
    email: user?.email ?? "",
    role: user?.role ?? ("user" as Role),
    password: "",
    // A password the administrator chose is a start password: by default the user replaces it.
    mustChangePassword: user?.mustChangePassword ?? true,
  })
  const feedback = {
    onSuccess: (u: UserView) => {
      toast.success(user ? `${u.username} saved` : `${u.username} created`, {
        description: user ? undefined : `Namespace ${u.namespace}`,
      })
      onClose()
    },
    onError: failed("save the user"),
  }
  const create = useCreateUser(feedback)
  const update = useUpdateUser(feedback)
  const pending = create.isPending || update.isPending
  const errors = fieldErrors(create.error ?? update.error)
  const { username, password, displayName, email, role, mustChangePassword } = draft
  // The identity provider sets profile and role of a linked account; an account without password gets none.
  const oidc = user?.oidc
  const passwordless = !!user && !user.hasPassword
  const tooShort = password !== "" && password.length < MIN_PASSWORD_LENGTH
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (user) {
      const changes = { displayName, email, role, mustChangePassword, ...(password ? { password } : {}) }
      update.mutate({ name: user.username, changes })
    } else {
      create.mutate({ username, password, displayName, email, role, mustChangePassword })
    }
  }

  return (
    <form onSubmit={submit}>
      <DialogHeader>
        <DialogTitle>{user ? `Edit ${user.username}` : "New user"}</DialogTitle>
        <DialogDescription>
          {oidc
            ? "Name, email and role come from the identity provider."
            : user
              ? "Leave the password empty to keep it. A new password signs the user out everywhere."
              : "The user gets an own namespace for game servers."}
        </DialogDescription>
      </DialogHeader>
      <FieldGroup className="py-4">
        {oidc && (
          <Field>
            <FieldLabel>Identity provider user ID</FieldLabel>
            <div className="flex items-center gap-2 rounded-lg bg-muted px-3 py-1.5">
              <code className="flex-1 truncate font-mono text-xs">{oidc.subject}</code>
              <CopyButton value={oidc.subject} label="Copy user ID" />
            </div>
            <FieldDescription className="break-all">{oidc.issuer}</FieldDescription>
          </Field>
        )}
        {!user && (
          <UsernameField
            id="u-name"
            value={username}
            onChange={(v) => set("username", v)}
            error={errors.username}
            placeholder="alice"
            autoFocus
          />
        )}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field>
            <FieldLabel htmlFor="u-display">Display name</FieldLabel>
            <Input
              id="u-display"
              value={displayName}
              disabled={!!oidc}
              onChange={(e) => set("displayName", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="u-role">Role</FieldLabel>
            <Select value={role} disabled={!!oidc} onValueChange={(v) => set("role", v as Role)}>
              <SelectTrigger id="u-role" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="admin">Administrator</SelectItem>
              </SelectContent>
            </Select>
          </Field>
        </div>
        <Field>
          <FieldLabel htmlFor="u-email">Email</FieldLabel>
          <Input
            id="u-email"
            type="email"
            value={email}
            disabled={!!oidc}
            onChange={(e) => set("email", e.target.value)}
          />
        </Field>
        {!passwordless && (
          <PasswordFields
            isNew={!user}
            password={password}
            onPassword={(v) => set("password", v)}
            mustChange={mustChangePassword}
            onMustChange={(v) => set("mustChangePassword", v)}
            tooShort={tooShort}
            error={errors.password}
          />
        )}
      </FieldGroup>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={pending || tooShort || (!user && (!username || !password))}>
          {pending && <Spinner />}
          {user ? "Save" : "Create user"}
        </Button>
      </DialogFooter>
    </form>
  )
}

/** Password (new or replacement) and whether the user must replace it after signing in. */
function PasswordFields({
  isNew,
  password,
  onPassword,
  mustChange,
  onMustChange,
  tooShort,
  error,
}: {
  isNew: boolean
  password: string
  onPassword: (v: string) => void
  mustChange: boolean
  onMustChange: (v: boolean) => void
  tooShort: boolean
  error?: string
}) {
  return (
    <>
      <Field data-invalid={tooShort || !!error}>
        <FieldLabel htmlFor="u-password">{isNew ? "Password" : "New password"}</FieldLabel>
        <UnveilPassword
          id="u-password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => onPassword(e.target.value)}
          aria-invalid={tooShort || !!error}
        />
        {error ? (
          <FieldError>{error}</FieldError>
        ) : (
          <FieldDescription>At least {MIN_PASSWORD_LENGTH} characters. Stored as Argon2id hash.</FieldDescription>
        )}
      </Field>
      <Field orientation="horizontal">
        <Switch id="u-must-change" checked={mustChange} onCheckedChange={onMustChange} />
        <FieldLabel htmlFor="u-must-change">Must choose a new password after signing in</FieldLabel>
      </Field>
    </>
  )
}
