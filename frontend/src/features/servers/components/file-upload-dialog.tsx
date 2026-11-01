import { useState } from "react"
import { UploadIcon } from "lucide-react"
import { toast } from "sonner"

import { FileDropzone } from "@/components/common/file-dropzone"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Spinner } from "@/components/ui/spinner"
import { uploadedText } from "@/features/servers/lib/files"
import { failed } from "@/lib/notify"
import { useUploadFiles } from "@/lib/queries"

/** Upload files into the current directory (several files at once); the dialog stays open while they upload. */
export function FileUploadDialog({
  open,
  server,
  dir,
  onClose,
}: {
  open: boolean
  server: string
  dir: string
  onClose: () => void
}) {
  // Kept outside the form, so closing is refused while the request runs.
  const upload = useUploadFiles(server, { onError: failed("upload the files") })
  return (
    <Dialog open={open} onOpenChange={(o) => !o && !upload.isPending && onClose()}>
      <DialogContent className="sm:max-w-xl" showCloseButton={!upload.isPending}>
        {/* mounted fresh on every open */}
        <UploadForm upload={upload} dir={dir} onClose={onClose} />
      </DialogContent>
    </Dialog>
  )
}

function UploadForm({
  upload,
  dir,
  onClose,
}: {
  upload: ReturnType<typeof useUploadFiles>
  dir: string
  onClose: () => void
}) {
  const [files, setFiles] = useState<File[]>([])
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault()
        if (!files.length) return
        upload.mutate(
          { dir, files },
          {
            onSuccess: () => {
              toast.success(uploadedText(files))
              onClose()
            },
          },
        )
      }}
    >
      <DialogHeader>
        <DialogTitle>Upload files</DialogTitle>
        <DialogDescription>
          Into <code className="font-mono">{dir}</code>. Add files (also one after another), then upload them together.
        </DialogDescription>
      </DialogHeader>
      <div className="max-h-[55vh] overflow-auto">
        <FileDropzone files={files} onChange={setFiles} />
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={upload.isPending}>
          Cancel
        </Button>
        <Button type="submit" disabled={!files.length || upload.isPending}>
          {upload.isPending ? <Spinner /> : <UploadIcon />}
          Upload {files.length > 0 && `(${files.length})`}
        </Button>
      </DialogFooter>
    </form>
  )
}
