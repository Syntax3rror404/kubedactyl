import { toast } from "sonner"

import { roles } from "@/app/config"
import { createdUserNote, fixedRole, newUserHint } from "@/app/users"
import { CopyField } from "@/base/components/copy-button"
import { OidcBadge } from "@/base/components/role-badge"
import { PasswordField } from "@/base/components/unveil-password"
import { UsernameField } from "@/base/components/username-field"
import { useDraft } from "@/base/hooks/use-draft"
import { roleLabel } from "@/base/lib/format"
import { failed, useCreateUser, useUpdateUser } from "@/base/lib/queries"
import type { Role, UserView } from "@/base/lib/types"
import { fieldErrors, MIN_PASSWORD_LENGTH, PASSWORD_RULE } from "@/base/lib/validation"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"

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
    role: user?.role ?? roles[0].value,
    password: "",
    // A password the administrator chose is a start password: by default the user replaces it.
    mustChangePassword: user?.mustChangePassword ?? true,
  })
  const onSuccess = (u: UserView) => {
    toast.success(user ? `${u.username} saved` : `${u.username} created`, {
      description: user ? undefined : createdUserNote(u),
    })
    onClose()
  }
  const create = useCreateUser({ onSuccess, onError: failed("create the user") })
  const update = useUpdateUser({ onSuccess, onError: failed("save the user") })
  const pending = create.isPending || update.isPending
  const errors = fieldErrors(create.error ?? update.error)
  const { username, password, displayName, email, role, mustChangePassword } = draft
  // The identity provider sets profile and role of a linked account (read only, marked with the OIDC badge); an
  // account without password gets none.
  const oidc = user?.oidc
  const fixed = user && fixedRole(user)
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
          {!user
            ? newUserHint
            : passwordless
              ? "Signs in through the identity provider."
              : "Leave the password empty to keep it. A new password signs the user out everywhere and revokes their API tokens."}
        </DialogDescription>
      </DialogHeader>
      <FieldGroup className="py-4">
        {oidc && (
          <Field>
            <FieldLabel>Identity provider user ID</FieldLabel>
            <CopyField value={oidc.subject} label="Copy user ID" />
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
            <FieldLabel htmlFor="u-display" className="items-center">
              Display name {oidc && <OidcBadge />}
            </FieldLabel>
            <Input
              id="u-display"
              value={displayName}
              readOnly={!!oidc}
              onChange={(e) => set("displayName", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="u-role" className="items-center">
              Role {oidc && <OidcBadge />}
            </FieldLabel>
            {oidc || fixed ? (
              <Input id="u-role" readOnly value={fixed ?? roleLabel(role)} />
            ) : (
              <RoleSelect id="u-role" value={role} onChange={(v) => set("role", v)} />
            )}
          </Field>
        </div>
        <Field>
          <FieldLabel htmlFor="u-email" className="items-center">
            Email {oidc && <OidcBadge />}
          </FieldLabel>
          <Input
            id="u-email"
            type="email"
            value={email}
            readOnly={!!oidc}
            onChange={(e) => set("email", e.target.value)}
          />
        </Field>
        {!passwordless && (
          <PasswordFields
            isNew={!user}
            password={password}
            onPasswordChange={(v) => set("password", v)}
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
  onPasswordChange,
  mustChange,
  onMustChange,
  tooShort,
  error,
}: {
  isNew: boolean
  password: string
  onPasswordChange: (v: string) => void
  mustChange: boolean
  onMustChange: (v: boolean) => void
  tooShort: boolean
  error?: string
}) {
  return (
    <>
      <PasswordField
        id="u-password"
        label={isNew ? "Password" : "New password"}
        autoComplete="new-password"
        value={password}
        onChange={onPasswordChange}
        invalid={tooShort}
        error={error}
        hint={PASSWORD_RULE}
      />
      <Field orientation="horizontal">
        <Switch id="u-must-change" checked={mustChange} onCheckedChange={onMustChange} />
        <FieldLabel htmlFor="u-must-change">Must choose a new password after signing in</FieldLabel>
      </Field>
    </>
  )
}

/** The role picker of the user and invite dialogs. */
export function RoleSelect({ id, value, onChange }: { id: string; value: Role; onChange: (role: Role) => void }) {
  return (
    <Select items={roles} value={value} onValueChange={(v) => v && onChange(v)}>
      <SelectTrigger id={id} className="w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {roles.map((r) => (
          <SelectItem key={r.value} value={r.value}>
            {r.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
