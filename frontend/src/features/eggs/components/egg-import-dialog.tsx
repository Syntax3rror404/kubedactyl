import { useState } from "react"
import { LinkIcon, PlusIcon, UploadIcon } from "lucide-react"
import { useNavigate } from "react-router"
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
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useImportEgg } from "@/lib/queries"

/** Imports an egg from a file or a URL. */
export function EggImportDialog() {
  const [open, setOpen] = useState(false)
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <PlusIcon />
          Import egg
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-xl">
        {/* mounted fresh on every open */}
        <ImportForm onClose={() => setOpen(false)} />
      </DialogContent>
    </Dialog>
  )
}

function ImportForm({ onClose }: { onClose: () => void }) {
  const { draft, set } = useDraft<{ tab: string; url: string; file: File | null }>({
    tab: "file",
    url: "",
    file: null,
  })
  const navigate = useNavigate()
  const mutation = useImportEgg({
    onSuccess: (egg) => {
      toast.success(`Egg “${egg.spec.displayName}” imported`, { description: `Format ${egg.spec.source?.format}` })
      onClose()
      navigate(`/eggs/${egg.metadata.name}`)
    },
    onError: failed("import the egg"),
  })
  const ready = draft.tab === "file" ? !!draft.file : /^https?:\/\//.test(draft.url)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (ready) mutation.mutate(draft.tab === "file" && draft.file ? { file: draft.file } : { url: draft.url })
  }

  return (
    <form onSubmit={submit} className="grid gap-4">
      <DialogHeader>
        <DialogTitle>Import egg</DialogTitle>
        <DialogDescription>
          JSON or YAML exports of Pterodactyl and Pelican panels. Importing an egg with an existing name updates it.
        </DialogDescription>
      </DialogHeader>
      <Tabs value={draft.tab} onValueChange={(v) => set("tab", v)}>
        <TabsList className="w-full">
          <TabsTrigger value="file">
            <UploadIcon />
            Upload file
          </TabsTrigger>
          <TabsTrigger value="url">
            <LinkIcon />
            From URL
          </TabsTrigger>
        </TabsList>
        <TabsContent value="file" className="pt-2">
          <FileDropzone
            files={draft.file ? [draft.file] : []}
            onChange={(files) => set("file", files[0] ?? null)}
            multiple={false}
            accept={{ "application/json": [".json"], "application/yaml": [".yaml", ".yml"] }}
            hint="Pterodactyl (PTDL) or Pelican (PLCN) egg, .json or .yaml"
          />
        </TabsContent>
        <TabsContent value="url" className="space-y-2 pt-2">
          <Label htmlFor="egg-url">Raw egg URL</Label>
          <Input
            id="egg-url"
            placeholder="https://raw.githubusercontent.com/pelican-eggs/minecraft/main/java/paper/egg-paper.yaml"
            value={draft.url}
            onChange={(e) => set("url", e.target.value)}
          />
          <p className="text-xs text-muted-foreground">The file is downloaded by the panel backend.</p>
        </TabsContent>
      </Tabs>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={!ready || mutation.isPending}>
          {mutation.isPending && <Spinner />}
          Import
        </Button>
      </DialogFooter>
    </form>
  )
}
