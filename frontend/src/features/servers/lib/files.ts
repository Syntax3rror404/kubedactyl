import type { FileEntry } from "@/lib/types"

/** Largest file the editor opens (same limit as the backend). */
const MAX_EDIT = 5 * 1024 * 1024

const binary =
  /\.(jar|zip|gz|tgz|xz|bz2|7z|rar|png|jpe?g|gif|webp|ico|mp3|ogg|wav|so|dll|exe|bin|dat|mca|mcr|nbt|db|sqlite|class|pak|vpk|bsp)$/i
const archive = /\.(zip|tar|tar\.gz|tgz|tar\.xz|txz|tar\.bz2|tbz2?)$/i

export const isArchive = (name: string) => archive.test(name)

/** Text files up to MAX_EDIT open in the editor; everything else is downloaded. */
export const isEditable = (e: FileEntry) => !e.isDirectory && !binary.test(e.name) && e.size <= MAX_EDIT

/** Drag and drop type for moving entries inside the file manager (JSON list of names). */
export const DRAG_TYPE = "application/x-kubedactyl-files"

/** Whether a drag event carries entries of the file manager (not files from the desktop). */
export const isFileDrag = (ev: React.DragEvent) => ev.dataTransfer.types.includes(DRAG_TYPE)

/** Names carried by a file manager drag. */
export const draggedNames = (ev: React.DragEvent): string[] => {
  try {
    return JSON.parse(ev.dataTransfer.getData(DRAG_TYPE) || "[]")
  } catch {
    return []
  }
}

export const joinPath = (dir: string, name: string) => (dir === "/" ? `/${name}` : `${dir}/${name}`)

/** Toast text after an upload: the file name, or how many files. */
export const uploadedText = (files: File[]) =>
  files.length === 1 ? `${files[0].name} uploaded` : `${files.length} files uploaded`
