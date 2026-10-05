import { useEffect, useState } from "react"
import { SaveIcon } from "lucide-react"
import { toast } from "sonner"

import { CodeEditor } from "@/components/common/code-editor"
import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Kbd } from "@/components/ui/kbd"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { languageFor } from "@/features/servers/lib/languages"
import { useBeforeUnload } from "@/hooks/use-before-unload"
import { failed } from "@/lib/notify"
import { useFileContent, useWriteFile } from "@/lib/queries"

/** Opens a file in CodeMirror; the editor mounts once the content has loaded. */
export function FileEditorDialog({
  server,
  file,
  isNew,
  onClose,
}: {
  server: string
  file: string
  isNew?: boolean
  onClose: () => void
}) {
  const content = useFileContent(server, file, !isNew)
  // Unsaved changes ask before the dialog closes (button, Esc, click outside) or the tab goes.
  const [dirty, setDirty] = useState(false)
  useBeforeUnload(dirty)
  const [asking, setAsking] = useState(false)
  const close = () => (dirty ? setAsking(true) : onClose())
  return (
    <Dialog open onOpenChange={(o) => !o && close()}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle className="font-mono text-base">{file}</DialogTitle>
          <DialogDescription>
            {dirty && <span className="mr-2 font-medium text-amber-600 dark:text-amber-400">● Unsaved changes ·</span>}
            Save with <Kbd>⌘</Kbd> <Kbd>S</Kbd> or <Kbd>Ctrl</Kbd> <Kbd>S</Kbd>
          </DialogDescription>
        </DialogHeader>
        <QueryState query={isNew ? { ...content, data: "" } : content} skeleton={<Skeleton className="h-[60vh]" />}>
          {(initial) => <EditorBody server={server} file={file} initial={initial} onClose={close} onDirty={setDirty} />}
        </QueryState>
      </DialogContent>
      <ConfirmDialog
        open={asking}
        onOpenChange={setAsking}
        title="Discard unsaved changes?"
        description={`The changes to ${file} are lost.`}
        confirmLabel="Discard"
        destructive
        onConfirm={onClose}
      />
    </Dialog>
  )
}

function EditorBody({
  server,
  file,
  initial,
  onClose,
  onDirty,
}: {
  server: string
  file: string
  initial: string
  onClose: () => void
  onDirty: (dirty: boolean) => void
}) {
  const [value, setValue] = useState(initial)
  const [saved, setSaved] = useState(initial)
  useEffect(() => onDirty(value !== saved), [value, saved, onDirty])
  const save = useWriteFile(server, {
    onSuccess: (_, { content }) => {
      setSaved(content)
      toast.success(`${file} saved`)
    },
    onError: failed("save"),
  })
  const { mutate } = save
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault()
        mutate({ file, content: value })
      }
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [mutate, file, value])

  return (
    <>
      <CodeEditor value={value} onChange={setValue} language={languageFor(file)} />
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>
          Close
        </Button>
        <Button disabled={save.isPending} onClick={() => save.mutate({ file, content: value })}>
          {save.isPending ? <Spinner /> : <SaveIcon />}
          Save
        </Button>
      </DialogFooter>
    </>
  )
}
