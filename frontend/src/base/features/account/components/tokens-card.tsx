import { useState } from "react"
import { KeySquareIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { apiTokenPrefix } from "@/app/config"
import { Callout } from "@/base/components/callout"
import { ConfirmDialog } from "@/base/components/confirm-dialog"
import { CopyField } from "@/base/components/copy-button"
import { EmptyState, QueryState } from "@/base/components/query-state"
import { useDraft } from "@/base/hooks/use-draft"
import { formatDate, formatRelativeTime, plural } from "@/base/lib/format"
import { failed, useCreateToken, useDeleteToken, useSettings, useTokens } from "@/base/lib/queries"
import type { CreatedToken } from "@/base/lib/types"
import { fieldErrors } from "@/base/lib/validation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"

const presets = [7, 30, 90, 365]
/** Longest token name (the panel checks the same limit). */
const MAX_TOKEN_NAME = 64

/** Lifetimes a new token can have: the presets below the panel's limit, plus the limit itself. */
const lifetimeOptions = (max: number) =>
  [...presets.filter((d) => d < max), max].map((d) => ({ value: d, label: dayLabel(d) }))
const dayLabel = (days: number) => (days % 365 === 0 ? plural(days / 365, "year") : plural(days, "day"))

/** Create and revoke API tokens (prefix apiTokenPrefix) for scripts. */
export function TokensCard() {
  const tokens = useTokens()
  const maxDays = useSettings().data?.apiTokenMaxDays ?? 90
  const { draft, set, reset } = useDraft({ name: "", days: 0 })
  const days = draft.days || Math.min(90, maxDays)
  const lifetimes = lifetimeOptions(maxDays)
  const [created, setCreated] = useState<CreatedToken | null>(null)
  const create = useCreateToken({
    onSuccess: (t) => {
      setCreated(t)
      reset()
    },
    onError: failed("create the token"),
  })
  const errors = fieldErrors(create.error)
  const revoke = useDeleteToken({
    onSuccess: () => toast.success("Token revoked"),
    onError: failed("revoke the token"),
  })
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    create.mutate({ name: draft.name.trim(), expiresInDays: days })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>API tokens</CardTitle>
        <CardDescription>
          For scripts and automation: <code className="font-mono text-xs">Authorization: Bearer {apiTokenPrefix}…</code>
          . Tokens have the same permissions as your account and last at most {dayLabel(maxDays)}.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <form onSubmit={submit} className="flex flex-col gap-2 sm:flex-row">
          <Input
            placeholder="Token name, e.g. ci"
            aria-label="Token name"
            maxLength={MAX_TOKEN_NAME}
            value={draft.name}
            onChange={(e) => set("name", e.target.value)}
            aria-invalid={errors.name ? true : undefined}
          />
          <Select items={lifetimes} value={days} onValueChange={(v) => v && set("days", v)}>
            <SelectTrigger
              className="sm:w-40"
              aria-label="Lifetime"
              aria-invalid={errors.expiresInDays ? true : undefined}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {lifetimes.map((d) => (
                <SelectItem key={d.value} value={d.value}>
                  {d.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button type="submit" disabled={!draft.name.trim() || create.isPending}>
            {create.isPending ? <Spinner /> : <PlusIcon />}
            Create
          </Button>
        </form>
        <FieldError>{errors.name ?? errors.expiresInDays}</FieldError>

        {created && (
          <Callout
            tone="success"
            title="Copy the token now, it will not be shown again."
            action={
              <Button variant="outline" size="sm" onClick={() => setCreated(null)}>
                Done
              </Button>
            }
          >
            <CopyField value={created.token} label="Copy token" />
          </Callout>
        )}

        <QueryState
          query={tokens}
          skeleton={<Skeleton className="h-16 rounded-xl" />}
          empty={<EmptyState variant="card" icon={<KeySquareIcon />} title="No tokens yet" />}
        >
          {(list) => (
            <div className="divide-y rounded-xl border">
              {list.map((t) => (
                <div key={t.id} className="flex items-center gap-3 p-3">
                  <KeySquareIcon className="size-4 text-muted-foreground" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{t.name}</div>
                    <div className="text-xs text-muted-foreground">
                      <code className="font-mono">
                        {apiTokenPrefix}
                        {t.id}_…
                      </code>{" "}
                      · created {formatRelativeTime(t.createdAt)} · expires {formatDate(t.expiresAt)}
                    </div>
                  </div>
                  <ConfirmDialog
                    trigger={
                      <Button variant="ghost" size="icon-sm" aria-label={`Revoke ${t.name}`}>
                        <Trash2Icon />
                      </Button>
                    }
                    title={`Revoke token “${t.name}”?`}
                    description="Scripts that use this token can no longer sign in. This cannot be undone."
                    confirmLabel="Revoke"
                    destructive
                    onConfirm={() => revoke.mutate(t.id)}
                  />
                </div>
              ))}
            </div>
          )}
        </QueryState>
      </CardContent>
    </Card>
  )
}
