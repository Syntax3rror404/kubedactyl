import { useState } from "react"
import { DownloadIcon, ExternalLinkIcon } from "lucide-react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { toast } from "sonner"

import { EggIcon } from "@/components/common/egg-icon"
import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { EggTabs } from "@/features/eggs/components/egg-tabs"
import { installedAs, repositoryName } from "@/features/eggs/lib/library"
import { failed } from "@/lib/notify"
import { useEggs, useImportEgg, useLibraryEgg } from "@/lib/queries"
import type { LibraryEggContent } from "@/lib/types"

/** /eggs/library/egg?repository=…&path=… (admins): an egg of the library before it is installed. */
export function LibraryEggPage() {
  const [params] = useSearchParams()
  const egg = useLibraryEgg(params.get("repository") ?? "", params.get("path") ?? "")
  return (
    <QueryState query={egg} skeleton={<Skeleton className="h-96 rounded-2xl" />}>
      {(e) => <LibraryEggDetail content={e} />}
    </QueryState>
  )
}

function LibraryEggDetail({ content: { egg, spec: s } }: { content: LibraryEggContent }) {
  const navigate = useNavigate()
  const installed = installedAs(useEggs().data, egg)
  // Until changed, the checkbox shows the setting of the installed egg (the egg list may load later).
  const [changed, setAutoUpdate] = useState<boolean>()
  const autoUpdate = changed ?? installed?.spec.source?.autoUpdate ?? false
  const install = useImportEgg({
    onSuccess: (e) => {
      toast.success(`Egg “${e.spec.displayName}” installed`)
      navigate(`/eggs/${e.metadata.name}`)
    },
    onError: failed("install the egg"),
  })
  return (
    <div className="space-y-8">
      <PageHeader
        icon={<EggIcon name={s.displayName} icon={s.icon} className="size-14 rounded-xl" />}
        title={s.displayName}
        description={
          <span className="flex flex-wrap items-center gap-2">
            <span>{s.author}</span>
            <Badge variant="outline" className="font-mono">
              {egg.format}
            </Badge>
            <a
              className="inline-flex items-center gap-1 font-mono text-xs underline-offset-4 hover:underline"
              href={`${egg.repository}/blob/HEAD/${egg.path}`}
              target="_blank"
              rel="noreferrer"
            >
              {repositoryName(egg.repository)}/{egg.path}
              <ExternalLinkIcon className="size-3" />
            </a>
            {installed && <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">Installed</Badge>}
          </span>
        }
        actions={
          <>
            <div className="flex items-center gap-2">
              <Checkbox id="auto-update" checked={autoUpdate} onCheckedChange={(v) => setAutoUpdate(v === true)} />
              <Label htmlFor="auto-update">Update automatically</Label>
            </div>
            {installed && (
              <Button variant="outline" asChild>
                <Link to={`/eggs/${installed.metadata.name}`}>Open installed egg</Link>
              </Button>
            )}
            <Button disabled={install.isPending} onClick={() => install.mutate({ url: egg.url, autoUpdate })}>
              {install.isPending ? <Spinner /> : <DownloadIcon />}
              {installed ? "Install again" : "Install"}
            </Button>
          </>
        }
      />
      <p className="text-sm text-muted-foreground">
        {installed ? "Installing again replaces the installed egg." : "The file's URL becomes the egg's update URL."}
      </p>
      {s.description && <p className="max-w-3xl whitespace-pre-line text-muted-foreground">{s.description}</p>}
      <EggTabs spec={s} />
    </div>
  )
}
