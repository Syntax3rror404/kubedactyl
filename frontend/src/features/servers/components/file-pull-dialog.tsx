import { useState } from "react"
import { LinkIcon } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { failed } from "@/lib/notify"
import { usePullFile } from "@/lib/queries"

/** Downloads a file from a URL into the current folder (runs in the file container). */
export function FilePullDialog({
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
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        {/* mounted fresh on every open */}
        <PullForm server={server} dir={dir} onClose={onClose} />
      </DialogContent>
    </Dialog>
  )
}

function PullForm({ server, dir, onClose }: { server: string; dir: string; onClose: () => void }) {
  const [url, setUrl] = useState("")
  const [filename, setFilename] = useState("")
  const valid = /^https?:\/\/\S+$/i.test(url.trim())
  const pull = usePullFile(server, {
    onSuccess: (job) => {
      toast.info(`Downloading ${job.label}`)
      onClose()
    },
    onError: failed("start the download"),
  })
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault()
        if (valid) pull.mutate({ url: url.trim(), dir, filename: filename.trim() })
      }}
    >
      <DialogHeader>
        <DialogTitle>Download from URL</DialogTitle>
        <DialogDescription>
          The file container downloads the file into <code className="font-mono">{dir}</code>, so large files do not go
          through your browser.
        </DialogDescription>
      </DialogHeader>
      <Field>
        <FieldLabel htmlFor="pull-url">URL</FieldLabel>
        <Input
          id="pull-url"
          autoFocus
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/plugin.jar"
          className="font-mono"
        />
        <FieldDescription>
          http or https only; private networks cannot be reached unless the administrator allows them.
        </FieldDescription>
      </Field>
      <Field>
        <FieldLabel htmlFor="pull-name">File name</FieldLabel>
        <Input
          id="pull-name"
          value={filename}
          onChange={(e) => setFilename(e.target.value)}
          placeholder="from the URL"
          className="font-mono"
        />
      </Field>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={!valid || pull.isPending}>
          {pull.isPending ? <Spinner /> : <LinkIcon />}
          Download
        </Button>
      </DialogFooter>
    </form>
  )
}
