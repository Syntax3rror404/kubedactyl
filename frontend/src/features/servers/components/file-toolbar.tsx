import {
  ArchiveIcon,
  FilePlusIcon,
  FolderPlusIcon,
  LinkIcon,
  RefreshCwIcon,
  Trash2Icon,
  UploadIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"

/** Buttons of the file manager: actions for the selection, reload, new folder/file, download from URL, upload. */
export function FileToolbar({
  ready,
  selected,
  refreshing,
  onRefresh,
  onCompress,
  onDelete,
  onCreate,
}: {
  ready: boolean
  selected: number
  refreshing: boolean
  onRefresh: () => void
  onCompress: () => void
  onDelete: () => void
  onCreate: (kind: "folder" | "newfile" | "pull" | "upload") => void
}) {
  return (
    <div className={`flex flex-wrap gap-2 ${ready ? "" : "pointer-events-none opacity-50"}`}>
      {selected > 0 && (
        <>
          <Button size="sm" variant="outline" onClick={onCompress}>
            <ArchiveIcon />
            Compress ({selected})
          </Button>
          <Button size="sm" variant="destructive" onClick={onDelete}>
            <Trash2Icon />
            Delete ({selected})
          </Button>
        </>
      )}
      <Button size="sm" variant="outline" onClick={onRefresh}>
        <RefreshCwIcon className={refreshing ? "animate-spin" : ""} />
      </Button>
      <Button size="sm" variant="outline" onClick={() => onCreate("folder")}>
        <FolderPlusIcon />
        Folder
      </Button>
      <Button size="sm" variant="outline" onClick={() => onCreate("newfile")}>
        <FilePlusIcon />
        File
      </Button>
      <Button size="sm" variant="outline" onClick={() => onCreate("pull")}>
        <LinkIcon />
        From URL
      </Button>
      <Button size="sm" onClick={() => onCreate("upload")}>
        <UploadIcon />
        Upload
      </Button>
    </div>
  )
}
