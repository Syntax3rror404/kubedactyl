import { useState } from "react"
import { FolderIcon, UploadIcon } from "lucide-react"
import { useOutletContext, useSearchParams } from "react-router"
import { toast } from "sonner"

import { EmptyState, QueryState } from "@/components/common/query-state"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { FileDeleteDialog } from "@/features/servers/components/file-delete-dialog"
import { FileEditorDialog } from "@/features/servers/components/file-editor-dialog"
import { FileNameDialog } from "@/features/servers/components/file-name-dialog"
import { FilePathBreadcrumb } from "@/features/servers/components/file-path-breadcrumb"
import { FilePullDialog } from "@/features/servers/components/file-pull-dialog"
import { FileTable, type FileActions } from "@/features/servers/components/file-table"
import { FileToolbar } from "@/features/servers/components/file-toolbar"
import { FileUploadDialog } from "@/features/servers/components/file-upload-dialog"
import { FilesSessionGate } from "@/features/servers/components/files-session-gate"
import { FilesStopCountdown } from "@/features/servers/components/files-stop-countdown"
import { JobProgressList } from "@/features/servers/components/job-progress-list"
import { useDesktopDrop } from "@/features/servers/hooks/use-desktop-drop"
import { useFileContainer } from "@/features/servers/hooks/use-file-container"
import { isEditable, joinPath, uploadedText } from "@/features/servers/lib/files"
import type { ServerContext } from "@/features/servers/server-layout"
import { urls } from "@/lib/api"
import { failed } from "@/lib/notify"
import {
  useCompressFiles,
  useCreateFolder,
  useDeleteFiles,
  useDecompressFile,
  useFileList,
  useRenameFiles,
  useRenameFile,
  useUploadFiles,
} from "@/lib/queries"
import type { FileEntry } from "@/lib/types"

type Dialog =
  | { kind: "folder" }
  | { kind: "newfile" }
  | { kind: "upload" }
  | { kind: "pull" }
  | { kind: "rename"; entry: FileEntry }
  | { kind: "delete"; names: string[] }
  | { kind: "edit"; file: string; isNew?: boolean }
  | null

/**
 * /servers/:server/files: file manager (all in the monospace font); the file container is started on
 * demand (see useFileContainer).
 */
export function FilesPage() {
  const { server } = useOutletContext<ServerContext>()
  const name = server.metadata.name
  const [params, setParams] = useSearchParams()
  const dir = params.get("dir") || "/"
  // The selection belongs to a directory; changing the directory clears it implicitly.
  const [selection, setSelection] = useState<{ dir: string; names: Set<string> }>({ dir: "/", names: new Set() })
  const [dialog, setDialog] = useState<Dialog>(null)

  const { files, ready, refresh, jobs } = useFileContainer(name)
  const list = useFileList(name, dir, ready)
  const selected = selection.dir === dir ? selection.names : new Set<string>()
  const setSelected = (names: Set<string>) => setSelection({ dir, names })
  const go = (d: string) => setParams(d === "/" ? {} : { dir: d })

  // Listings reload in the hooks; failures are reported the same way for every operation.
  const onError = failed("change the files")
  const uploadFiles = useUploadFiles(name, { onError })
  const moveFiles = useRenameFiles(name, { onError })
  const renameFile = useRenameFile(name, { onError })
  const createFolder = useCreateFolder(name, { onError })
  const compressFiles = useCompressFiles(name, { onError, onSuccess: (r) => toast.success(`${r.name} created`) })
  const decompressFile = useDecompressFile(name, { onError, onSuccess: (_, v) => toast.success(`${v.name} extracted`) })
  const deleteFiles = useDeleteFiles(name, { onError, onSuccess: () => setSelected(new Set()) })
  // Files dragged from the desktop are uploaded into the current folder.
  const drop = useDesktopDrop(ready, (dropped) =>
    uploadFiles.mutate({ dir, files: dropped }, { onSuccess: () => toast.success(uploadedText(dropped)) }),
  )
  const move = (names: string[], target: string) => {
    if (target === dir || names.length === 0) return
    const moves = names.map((n) => ({ from: joinPath(dir, n), to: joinPath(target, n) }))
    moveFiles.mutate(moves, {
      onSuccess: () => {
        setSelected(new Set())
        toast.success(
          names.length === 1 ? `${names[0]} moved to ${target}` : `${names.length} items moved to ${target}`,
        )
      },
    })
  }
  const compress = (names: string[]) => compressFiles.mutate({ dir, names })

  const actions: FileActions = {
    open: (e) => {
      if (e.isDirectory) return go(joinPath(dir, e.name))
      if (!isEditable(e)) return window.location.assign(urls.fileDownload(name, joinPath(dir, e.name)))
      setDialog({ kind: "edit", file: joinPath(dir, e.name) })
    },
    edit: (e) => setDialog({ kind: "edit", file: joinPath(dir, e.name) }),
    downloadUrl: (e) => urls.fileDownload(name, joinPath(dir, e.name)),
    rename: (e) => setDialog({ kind: "rename", entry: e }),
    decompress: (e) => decompressFile.mutate({ dir, name: e.name }),
    compress: (e) => compress([e.name]),
    delete: (e) => setDialog({ kind: "delete", names: [e.name] }),
  }
  const items = list.data?.items ?? []

  return (
    <div className="space-y-4 font-mono">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <FilePathBreadcrumb dir={dir} onNavigate={go} onDropEntries={ready ? move : undefined} />
        <FileToolbar
          ready={ready}
          selected={selected.size}
          refreshing={list.isFetching}
          onRefresh={refresh}
          onCompress={() => compress([...selected])}
          onDelete={() => setDialog({ kind: "delete", names: [...selected] })}
          onCreate={(kind) => setDialog({ kind })}
        />
      </div>

      <JobProgressList jobs={jobs.data} kinds={["pull"]} limit={3} />

      <FilesSessionGate files={files}>
        <Card className="relative py-0" {...drop.handlers}>
          {drop.dropping && (
            <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-xl border-2 border-dashed border-sky-500 bg-sky-500/10 text-sm font-medium text-sky-700 backdrop-blur-[1px] dark:text-sky-300">
              <UploadIcon className="mr-2 size-4" />
              Drop to upload into {dir}
            </div>
          )}
          <QueryState
            query={list}
            skeleton={
              <div className="space-y-2 p-4">
                {Array.from({ length: 6 }).map((_, i) => (
                  <Skeleton key={i} className="h-8" />
                ))}
              </div>
            }
            isEmpty={(data) => data.items.length === 0}
            empty={
              <EmptyState
                variant="card"
                icon={<FolderIcon />}
                title="This folder is empty"
                description="Upload files or create a new one."
              />
            }
          >
            {(data) => (
              <FileTable
                items={data.items}
                dir={dir}
                selected={selected}
                onSelect={setSelected}
                actions={actions}
                onMove={move}
              />
            )}
          </QueryState>
        </Card>
      </FilesSessionGate>

      <FilesStopCountdown session={files.session} updatedAt={files.updatedAt} />
      {ready && items.length > 0 && (
        <p className="text-xs text-muted-foreground">
          Tip: drag files onto a folder to move them, or drop files here to upload.
        </p>
      )}

      <FileNameDialog
        open={dialog?.kind === "folder"}
        title="New folder"
        label="Folder name"
        onClose={() => setDialog(null)}
        pending={createFolder.isPending}
        onSubmit={(n) => createFolder.mutate({ dir, name: n }, { onSuccess: () => setDialog(null) })}
      />
      <FileNameDialog
        open={dialog?.kind === "newfile"}
        title="New file"
        label="File name"
        onClose={() => setDialog(null)}
        onSubmit={(n) => setDialog({ kind: "edit", file: joinPath(dir, n), isNew: true })}
      />
      <FileNameDialog
        open={dialog?.kind === "rename"}
        title="Rename or move"
        label="New name (use / to move into a folder)"
        initial={dialog?.kind === "rename" ? dialog.entry.name : ""}
        onClose={() => setDialog(null)}
        pending={renameFile.isPending}
        onSubmit={(n) =>
          dialog?.kind === "rename" &&
          renameFile.mutate({ dir, from: dialog.entry.name, to: n }, { onSuccess: () => setDialog(null) })
        }
      />
      <FileUploadDialog open={dialog?.kind === "upload"} server={name} dir={dir} onClose={() => setDialog(null)} />
      <FilePullDialog open={dialog?.kind === "pull"} server={name} dir={dir} onClose={() => setDialog(null)} />
      <FileDeleteDialog
        names={dialog?.kind === "delete" ? dialog.names : null}
        onClose={() => setDialog(null)}
        onConfirm={(names) => deleteFiles.mutate({ dir, names })}
      />
      {dialog?.kind === "edit" && (
        <FileEditorDialog
          server={name}
          file={dialog.file}
          isNew={dialog.isNew}
          onClose={() => {
            setDialog(null)
            refresh()
          }}
        />
      )}
    </div>
  )
}
