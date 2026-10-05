import { useState } from "react"

import { isFileDrag } from "@/features/servers/lib/files"

/**
 * Drop zone for files dragged from the desktop (not entries moved inside the file table): spread
 * `handlers` on the element; `dropping` is true while files hover over it.
 */
export function useDesktopDrop(enabled: boolean, onDrop: (files: File[]) => void) {
  const [dropping, setDropping] = useState(false)
  const isDesktopDrag = (e: React.DragEvent) => e.dataTransfer.types.includes("Files") && !isFileDrag(e)
  const handlers = {
    onDragOver: (e: React.DragEvent) => {
      if (!enabled || !isDesktopDrag(e)) return
      e.preventDefault()
      e.dataTransfer.dropEffect = "copy"
      setDropping(true)
    },
    onDragLeave: (e: React.DragEvent) => {
      if (!e.currentTarget.contains(e.relatedTarget as Node)) setDropping(false)
    },
    onDrop: (e: React.DragEvent) => {
      if (!isDesktopDrag(e)) return
      e.preventDefault()
      setDropping(false)
      const files = [...e.dataTransfer.files]
      if (files.length) onDrop(files)
    },
  }
  return { dropping, handlers }
}
