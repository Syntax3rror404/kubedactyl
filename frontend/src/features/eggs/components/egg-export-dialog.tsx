import { useState } from "react"
import { DownloadIcon, FileDownIcon } from "lucide-react"

import { CodeEditor } from "@/components/common/code-editor"
import { CopyButton } from "@/components/common/copy-button"
import { QueryState } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { urls } from "@/lib/api"
import { useEggExport } from "@/lib/queries"
import type { EggExportFormat } from "@/lib/types"

const formats: { id: EggExportFormat; label: string; hint: string }[] = [
  { id: "yaml", label: "YAML", hint: "Pelican format (PLCN_v3). Import into Pelican or Kubedactyl." },
  { id: "json", label: "JSON", hint: "The same Pelican egg (PLCN_v3) as JSON." },
  {
    id: "ptdl",
    label: "Pterodactyl",
    hint: "PTDL_v2 JSON for Pterodactyl, without UUID, tags and icon.",
  },
]

/** Export preview with a format switch; the panel generates the file. */
export function EggExportDialog({ name }: { name: string }) {
  const [open, setOpen] = useState(false)
  const [format, setFormat] = useState<EggExportFormat>("yaml")
  const text = useEggExport(name, format, open)
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <FileDownIcon />
          Export
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Export egg</DialogTitle>
          <DialogDescription>{formats.find((f) => f.id === format)!.hint}</DialogDescription>
        </DialogHeader>
        <Tabs value={format} onValueChange={(v) => setFormat(v as EggExportFormat)}>
          <TabsList>
            {formats.map((f) => (
              <TabsTrigger key={f.id} value={f.id}>
                {f.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <QueryState query={text} skeleton={<Skeleton className="h-[50vh] rounded-lg" />}>
          {(data) => <CodeEditor value={data} language={format === "yaml" ? "yaml" : "json"} readOnly height="50vh" />}
        </QueryState>
        <div className="flex justify-end gap-2">
          {text.data && <CopyButton value={text.data} />}
          <Button asChild>
            <a href={urls.eggExport(name, format)} download>
              <DownloadIcon />
              Download
            </a>
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
