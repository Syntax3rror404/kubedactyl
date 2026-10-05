import { useState } from "react"
import { ArchiveIcon, ArchiveRestoreIcon, DownloadIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { useOutletContext } from "react-router"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { EmptyState, QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { FilesSessionGate } from "@/features/servers/components/files-session-gate"
import { JobProgressList } from "@/features/servers/components/job-progress-list"
import { useFileContainer } from "@/features/servers/hooks/use-file-container"
import type { ServerContext } from "@/features/servers/server-layout"
import { urls } from "@/lib/api"
import { formatBytes, formatRelativeTime, phaseOf, plural, stoppedPhases } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useBackups, useCreateBackup, useDeleteBackup, useRestoreBackup } from "@/lib/queries"
import type { Backup } from "@/lib/types"

/** Backups: tar.gz archives of all server files in the .backups folder of the volume. */
export function BackupsPage() {
  const { server } = useOutletContext<ServerContext>()
  const name = server.metadata.name
  const { files, ready, jobs } = useFileContainer(name)
  const backups = useBackups(name, ready)
  const [label, setLabel] = useState("")
  // One state for the dialogs of the page.
  const [dialog, setDialog] = useState<{ kind: "restore" | "delete"; backup: Backup } | null>(null)

  const busy = jobs.data?.some((j) => j.state === "running" && (j.kind === "backup" || j.kind === "restore"))
  const stopped = stoppedPhases.includes(phaseOf(server))

  const create = useCreateBackup(name, {
    onSuccess: () => setLabel(""),
    onError: failed("create the backup"),
  })
  const restore = useRestoreBackup(name, {
    onError: failed("restore the backup"),
  })
  const remove = useDeleteBackup(name, {
    onSuccess: (_, backup) => toast.success(`Backup ${backup} deleted`),
    onError: failed("delete the backup"),
  })
  const total = (backups.data ?? []).reduce((sum, b) => sum + b.size, 0)

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ArchiveIcon className="size-4" />
            Backups
          </CardTitle>
          <CardDescription>
            Archives of all server files, stored on the server volume (they count towards its disk size). Restoring
            replaces all files.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-2 sm:flex-row"
            onSubmit={(e) => {
              e.preventDefault()
              create.mutate(label.trim())
            }}
          >
            <Input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Label (optional), e.g. before-update"
              maxLength={40}
              className="sm:max-w-xs"
              aria-label="Backup label"
            />
            <Button type="submit" disabled={create.isPending || busy}>
              {create.isPending ? <Spinner /> : <PlusIcon />}
              Create backup
            </Button>
          </form>
        </CardContent>
      </Card>

      <JobProgressList jobs={jobs.data} kinds={["backup", "restore"]} />

      <FilesSessionGate files={files}>
        <QueryState
          query={backups}
          skeleton={<Skeleton className="h-40 rounded-2xl" />}
          empty={
            <Card className="py-0">
              <EmptyState
                variant="card"
                icon={<ArchiveIcon />}
                title="No backups yet"
                description="Create one above or add a “Create backup” task to a schedule."
              />
            </Card>
          }
        >
          {(items) => (
            <Card className="py-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pl-4">Backup</TableHead>
                    <TableHead className="text-right">Size</TableHead>
                    <TableHead className="hidden sm:table-cell">Created</TableHead>
                    <TableHead className="pr-4 text-right">
                      <span className="sr-only">Actions</span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((b) => (
                    <TableRow key={b.name}>
                      <TableCell className="max-w-0 truncate pl-4 font-mono text-xs sm:max-w-none">{b.name}</TableCell>
                      <TableCell className="text-right tabular-nums">{formatBytes(b.size)}</TableCell>
                      <TableCell
                        className="hidden text-muted-foreground sm:table-cell"
                        title={new Date(b.createdAt).toLocaleString()}
                      >
                        {formatRelativeTime(b.createdAt)}
                      </TableCell>
                      <TableCell className="pr-4">
                        <div className="flex justify-end gap-1">
                          <Button size="icon-sm" variant="ghost" asChild title="Download">
                            <a href={urls.fileDownload(name, b.path)} download>
                              <DownloadIcon />
                            </a>
                          </Button>
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            title="Restore"
                            disabled={busy}
                            onClick={() => setDialog({ kind: "restore", backup: b })}
                          >
                            <ArchiveRestoreIcon />
                          </Button>
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            title="Delete"
                            className="text-destructive"
                            onClick={() => setDialog({ kind: "delete", backup: b })}
                          >
                            <Trash2Icon />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <p className="border-t px-4 py-2 text-xs text-muted-foreground">
                {plural(items.length, "backup")} · {formatBytes(total)} in total
              </p>
            </Card>
          )}
        </QueryState>
      </FilesSessionGate>

      <ConfirmDialog
        open={dialog?.kind === "restore"}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Restore ${dialog?.backup.name}?`}
        description={
          stopped
            ? "All server files except backups are replaced by this backup. The server cannot start until it is done."
            : "Stop the server first, files cannot be restored while it runs."
        }
        confirmLabel="Delete files and restore"
        destructive
        disabled={!stopped}
        onConfirm={() => dialog && restore.mutate(dialog.backup.name)}
      />
      <ConfirmDialog
        open={dialog?.kind === "delete"}
        onOpenChange={(o) => !o && setDialog(null)}
        title={`Delete ${dialog?.backup.name}?`}
        description="The archive is removed from the server volume. This cannot be undone."
        confirmLabel="Delete"
        destructive
        onConfirm={() => dialog && remove.mutate(dialog.backup.name)}
      />
    </div>
  )
}
