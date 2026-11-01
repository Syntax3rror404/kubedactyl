import { useState } from "react"
import { CopyPlusIcon, EllipsisIcon, PencilIcon, PlusIcon, RefreshCwIcon, Trash2Icon } from "lucide-react"
import { Link, useNavigate, useParams } from "react-router"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { EggExportDialog } from "@/features/eggs/components/egg-export-dialog"
import { EggHeader } from "@/features/eggs/components/egg-header"
import { EggOrigin } from "@/features/eggs/components/egg-origin"
import { EggTabs } from "@/features/eggs/components/egg-tabs"
import { UpdateUrlBadge } from "@/features/eggs/components/update-url-badge"
import { formatRelativeTime } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useDeleteEgg, useEgg, useUpdateEggFromUrl } from "@/lib/queries"
import type { Egg } from "@/lib/types"

/** /eggs/:egg (admins): the egg tabs plus edit, export, duplicate, update and delete. */
export function EggDetailPage() {
  const { egg: name = "" } = useParams()
  const egg = useEgg(name)
  return (
    <QueryState query={egg} skeleton={<Skeleton className="h-96 rounded-2xl" />}>
      {(e) => <EggDetail egg={e} />}
    </QueryState>
  )
}

type Dialog = { kind: "update" } | { kind: "delete" } | null

function EggDetail({ egg }: { egg: Egg }) {
  const name = egg.metadata.name
  const navigate = useNavigate()
  const [dialog, setDialog] = useState<Dialog>(null)
  const s = egg.spec
  const update = useUpdateEggFromUrl(name, {
    onSuccess: (e) => toast.success(`Egg “${e.spec.displayName}” updated`),
    onError: failed("update the egg"),
  })
  const del = useDeleteEgg(name, {
    onSuccess: () => {
      toast.success(`Egg “${s.displayName}” deleted`)
      navigate("/eggs")
    },
    onError: failed("delete the egg"),
  })

  return (
    <div className="space-y-8">
      <EggHeader
        spec={s}
        actions={
          <>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label="More actions">
                  <EllipsisIcon />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem asChild>
                  <Link to={`/eggs/new?from=${name}`}>
                    <CopyPlusIcon />
                    Duplicate
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!s.source?.updateUrl || update.isPending}
                  onClick={() => setDialog({ kind: "update" })}
                >
                  <RefreshCwIcon />
                  Update from URL
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem variant="destructive" onClick={() => setDialog({ kind: "delete" })}>
                  <Trash2Icon />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <EggExportDialog name={name} />
            <Button variant="outline" asChild>
              <Link to={`/eggs/${name}/edit`}>
                <PencilIcon />
                Edit
              </Link>
            </Button>
            <Button asChild>
              <Link to={`/servers/new?egg=${name}`}>
                <PlusIcon />
                Create server
              </Link>
            </Button>
          </>
        }
      >
        <EggOrigin spec={s} className="font-mono" />
        <UpdateUrlBadge spec={s} />
        {s.source?.editedAt && <span className="text-xs">edited {formatRelativeTime(s.source.editedAt)}</span>}
        {s.source?.autoUpdate && egg.status?.updateCheckedAt && (
          <span className={egg.status.updateError ? "text-xs text-destructive" : "text-xs"}>
            {egg.status.updateError
              ? `auto update failed ${formatRelativeTime(egg.status.updateCheckedAt)}: ${egg.status.updateError}`
              : `checked ${formatRelativeTime(egg.status.updateCheckedAt)}` +
                (egg.status.updatedAt ? ` · updated ${formatRelativeTime(egg.status.updatedAt)}` : "")}
          </span>
        )}
      </EggHeader>

      <ConfirmDialog
        open={dialog?.kind === "update"}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Update ${s.displayName} from its URL?`}
        description={
          <>
            Downloads <span className="font-mono break-all">{s.source?.updateUrl}</span> and replaces all fields:
            changes made in the panel are overwritten.
          </>
        }
        confirmLabel="Update"
        onConfirm={() => update.mutate()}
      />
      <ConfirmDialog
        open={dialog?.kind === "delete"}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Delete ${s.displayName}?`}
        description="Eggs that are used by servers cannot be deleted."
        confirmLabel="Delete"
        destructive
        onConfirm={() => del.mutate()}
      />

      <EggTabs spec={s} />
    </div>
  )
}
