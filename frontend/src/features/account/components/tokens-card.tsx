import { useState } from "react"
import { KeySquareIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { Callout } from "@/components/common/callout"
import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { CopyButton } from "@/components/common/copy-button"
import { EmptyState, QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { useDraft } from "@/hooks/use-draft"
import { formatDate, formatRelativeTime, plural } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useCreateToken, useDeleteToken, useSettings, useTokens } from "@/lib/queries"
import type { CreatedToken } from "@/lib/types"

const presets = [7, 30, 90, 365]

/** Lifetimes a new token can have: the presets below the panel's limit, plus the limit itself. */
const lifetimeOptions = (max: number) => [...presets.filter((d) => d < max), max]
const dayLabel = (days: number) => (days % 365 === 0 ? plural(days / 365, "year") : plural(days, "day"))

/** Create and revoke API tokens (kdt_…) for scripts. */
export function TokensCard() {
  const tokens = useTokens()
  const maxDays = useSettings().data?.apiTokenMaxDays ?? 90
  const { draft, set, reset } = useDraft({ name: "", days: "" })
  const days = draft.days || String(Math.min(90, maxDays))
  const [created, setCreated] = useState<CreatedToken | null>(null)
  const create = useCreateToken({
    onSuccess: (t) => {
      setCreated(t)
      reset()
    },
    onError: failed("create the token"),
  })
  const revoke = useDeleteToken({
    onSuccess: () => toast.success("Token revoked"),
    onError: failed("revoke the token"),
  })
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    create.mutate({ name: draft.name.trim(), expiresInDays: Number(days) })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>API tokens</CardTitle>
        <CardDescription>
          For scripts and automation: <code className="font-mono text-xs">Authorization: Bearer kdt_…</code>. Tokens
          have the same permissions as your account and last at most {dayLabel(maxDays)}.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <form onSubmit={submit} className="flex flex-col gap-2 sm:flex-row">
          <Input placeholder="Token name, e.g. ci" value={draft.name} onChange={(e) => set("name", e.target.value)} />
          <Select value={days} onValueChange={(v) => set("days", v)}>
            <SelectTrigger className="sm:w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {lifetimeOptions(maxDays).map((d) => (
                <SelectItem key={d} value={String(d)}>
                  {dayLabel(d)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button type="submit" disabled={!draft.name.trim() || create.isPending}>
            {create.isPending ? <Spinner /> : <PlusIcon />}
            Create
          </Button>
        </form>

        {created && (
          <Callout tone="success" title="Copy the token now, it will not be shown again.">
            <div className="flex items-center gap-2 rounded-lg bg-muted px-3 py-1.5">
              <code className="flex-1 truncate font-mono text-xs">{created.token}</code>
              <CopyButton value={created.token} label="Copy token" />
            </div>
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
                      <code className="font-mono">kdt_{t.id}_…</code> · created {formatRelativeTime(t.createdAt)} ·{" "}
                      expires {formatDate(t.expiresAt)}
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
