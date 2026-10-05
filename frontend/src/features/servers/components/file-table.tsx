import { useState } from "react"
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  DownloadIcon,
  FileCodeIcon,
  FileIcon,
  FolderIcon,
  MoreHorizontalIcon,
  PencilIcon,
  Trash2Icon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { DRAG_TYPE, draggedNames, isArchive, isEditable, isFileDrag, joinPath } from "@/features/servers/lib/files"
import { languageFor } from "@/features/servers/lib/languages"
import { formatBytes, formatRelativeTime } from "@/lib/format"
import type { FileEntry } from "@/lib/types"
import { cn } from "@/lib/utils"

export interface FileActions {
  open: (e: FileEntry) => void
  edit: (e: FileEntry) => void
  downloadUrl: (e: FileEntry) => string
  rename: (e: FileEntry) => void
  decompress: (e: FileEntry) => void
  compress: (e: FileEntry) => void
  delete: (e: FileEntry) => void
}

/** Directory listing with selection, row actions and drag and drop onto folders. */
export function FileTable({
  items,
  dir,
  selected,
  onSelect,
  actions,
  onMove,
}: {
  items: FileEntry[]
  dir: string
  selected: Set<string>
  onSelect: (s: Set<string>) => void
  actions: FileActions
  /** Moves entries of this directory into another directory (absolute path). */
  onMove: (names: string[], target: string) => void
}) {
  // Entries being dragged and the folder under the pointer.
  const [dragging, setDragging] = useState<string[] | null>(null)
  const [over, setOver] = useState<string | null>(null)
  const allSelected = items.length > 0 && selected.size === items.length
  const toggle = (name: string, on: boolean) => {
    const next = new Set(selected)
    if (on) next.add(name)
    else next.delete(name)
    onSelect(next)
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-10 pl-4">
            <Checkbox
              checked={allSelected}
              onCheckedChange={(c) => onSelect(c ? new Set(items.map((i) => i.name)) : new Set())}
            />
          </TableHead>
          <TableHead>Name</TableHead>
          <TableHead className="w-28 text-right">Size</TableHead>
          <TableHead className="hidden w-36 text-right md:table-cell">Modified</TableHead>
          <TableHead className="w-12" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((e) => (
          <TableRow
            key={e.name}
            className={cn(
              "group cursor-pointer",
              dragging?.includes(e.name) && "opacity-50",
              over === e.name && "bg-sky-500/10 ring-1 ring-sky-500 ring-inset",
            )}
            data-state={selected.has(e.name) ? "selected" : undefined}
            onClick={() => actions.open(e)}
            draggable
            onDragStart={(ev) => {
              // Dragging a selected entry moves the whole selection.
              const names = selected.has(e.name) ? [...selected] : [e.name]
              ev.dataTransfer.setData(DRAG_TYPE, JSON.stringify(names))
              ev.dataTransfer.effectAllowed = "move"
              setDragging(names)
            }}
            onDragEnd={() => (setDragging(null), setOver(null))}
            {...(e.isDirectory && {
              onDragOver: (ev: React.DragEvent) => {
                if (!isFileDrag(ev) || dragging?.includes(e.name)) return
                ev.preventDefault()
                ev.dataTransfer.dropEffect = "move"
                setOver(e.name)
              },
              onDragLeave: () => setOver((cur) => (cur === e.name ? null : cur)),
              onDrop: (ev: React.DragEvent) => {
                ev.preventDefault()
                setOver(null)
                onMove(
                  draggedNames(ev).filter((n) => n !== e.name),
                  joinPath(dir, e.name),
                )
              },
            })}
          >
            <TableCell className="pl-4" onClick={(ev) => ev.stopPropagation()}>
              <Checkbox checked={selected.has(e.name)} onCheckedChange={(c) => toggle(e.name, !!c)} />
            </TableCell>
            <TableCell>
              <span className="flex items-center gap-2.5">
                <EntryIcon entry={e} />
                <span className="font-medium">{e.name}</span>
                {e.isSymlink && <span className="text-xs text-muted-foreground">symlink</span>}
              </span>
            </TableCell>
            <TableCell className="text-right text-xs text-muted-foreground tabular-nums">
              {e.isDirectory ? "" : formatBytes(e.size)}
            </TableCell>
            <TableCell className="hidden text-right text-xs text-muted-foreground md:table-cell">
              {formatRelativeTime(e.modifiedAt)}
            </TableCell>
            <TableCell onClick={(ev) => ev.stopPropagation()}>
              <RowMenu entry={e} actions={actions} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function EntryIcon({ entry }: { entry: FileEntry }) {
  if (entry.isDirectory) return <FolderIcon className="size-4 fill-sky-500/20 text-sky-500" />
  if (languageFor(entry.name)) return <FileCodeIcon className="size-4 text-emerald-500" />
  if (isArchive(entry.name)) return <ArchiveIcon className="size-4 text-amber-500" />
  return <FileIcon className="size-4 text-muted-foreground" />
}

function RowMenu({ entry: e, actions }: { entry: FileEntry; actions: FileActions }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon-sm" className="opacity-60 group-hover:opacity-100">
          <MoreHorizontalIcon />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {isEditable(e) && (
          <DropdownMenuItem onClick={() => actions.edit(e)}>
            <FileCodeIcon />
            Edit
          </DropdownMenuItem>
        )}
        {!e.isDirectory && (
          <DropdownMenuItem asChild>
            <a href={actions.downloadUrl(e)}>
              <DownloadIcon />
              Download
            </a>
          </DropdownMenuItem>
        )}
        <DropdownMenuItem onClick={() => actions.rename(e)}>
          <PencilIcon />
          Rename / move
        </DropdownMenuItem>
        {isArchive(e.name) && (
          <DropdownMenuItem onClick={() => actions.decompress(e)}>
            <ArchiveRestoreIcon />
            Extract here
          </DropdownMenuItem>
        )}
        <DropdownMenuItem onClick={() => actions.compress(e)}>
          <ArchiveIcon />
          Compress
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onClick={() => actions.delete(e)}>
          <Trash2Icon />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
