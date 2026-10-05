import { FileIcon, UploadCloudIcon, XIcon } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"
import { useDropzone, type Accept } from "react-dropzone"

import { Button } from "@/components/ui/button"
import { formatBytes } from "@/lib/format"
import { cn } from "@/lib/utils"

/**
 * Drop zone for files: click or drag files onto it. Controlled: the caller keeps the list.
 * With multiple=false a new file replaces the selected one.
 */
export function FileDropzone({
  files,
  onChange,
  multiple = true,
  accept,
  hint,
}: {
  files: File[]
  onChange: (files: File[]) => void
  multiple?: boolean
  accept?: Accept
  hint?: string
}) {
  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    multiple,
    accept,
    noClick: true,
    onDrop: (dropped) =>
      onChange(
        multiple
          ? [...files, ...dropped.filter((d) => !files.some((f) => f.name === d.name && f.size === d.size))]
          : dropped.slice(0, 1),
      ),
  })
  return (
    <div className="space-y-3">
      <div
        {...getRootProps()}
        onClick={open}
        className={cn(
          "group relative flex cursor-pointer flex-col items-center justify-center gap-2 overflow-hidden rounded-xl border border-dashed px-6 py-10 text-center transition-colors",
          isDragActive ? "border-primary bg-primary/5" : "hover:border-foreground/30 hover:bg-muted/40",
        )}
      >
        <input {...getInputProps()} />
        <motion.div
          animate={{ y: isDragActive ? -4 : 0, scale: isDragActive ? 1.08 : 1 }}
          transition={{ type: "spring", stiffness: 300, damping: 20 }}
          className="rounded-full border bg-background p-3 shadow-sm"
        >
          <UploadCloudIcon
            className={cn("size-6 text-muted-foreground transition-colors", isDragActive && "text-primary")}
          />
        </motion.div>
        <p className="text-sm font-medium">
          {isDragActive
            ? "Drop to add"
            : multiple
              ? "Drop files here or click to choose"
              : "Drop a file here or click to choose"}
        </p>
        {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
      </div>
      <AnimatePresence initial={false}>
        {files.map((f, i) => (
          <motion.div
            key={`${f.name}-${f.size}-${f.lastModified}`}
            layout
            initial={{ opacity: 0, y: -6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, height: 0 }}
            className="flex items-center gap-3 rounded-lg border px-3 py-2 text-sm"
          >
            <FileIcon className="size-4 shrink-0 text-muted-foreground" />
            <span className="min-w-0 flex-1 truncate font-mono text-xs">{f.name}</span>
            <span className="shrink-0 text-xs text-muted-foreground tabular-nums">{formatBytes(f.size)}</span>
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              title="Remove"
              onClick={() => onChange(files.filter((_, j) => j !== i))}
            >
              <XIcon />
            </Button>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  )
}
