import { useState } from "react"
import { DownloadIcon, ExternalLinkIcon } from "lucide-react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { toast } from "sonner"

import { QueryState } from "@/components/common/query-state"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { EggHeader } from "@/features/eggs/components/egg-header"
import { EggTabs } from "@/features/eggs/components/egg-tabs"
import { installedAs, repositoryName } from "@/features/eggs/lib/library"
import { failed } from "@/lib/notify"
import { useEggs, useImportEgg, useLibraryEgg } from "@/lib/queries"
import type { LibraryEggContent } from "@/lib/types"
import { cn } from "@/lib/utils"

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
  // Installing again keeps auto update on (it is turned off in the editor), so the box only turns it on.
  const keepsAutoUpdate = !!installed?.spec.source?.autoUpdate
  const [chosen, setAutoUpdate] = useState(false)
  const autoUpdate = keepsAutoUpdate || chosen
  const install = useImportEgg({
    onSuccess: (e) => {
      toast.success(`Egg “${e.spec.displayName}” installed`)
      navigate(`/eggs/${e.metadata.name}`)
    },
    onError: failed("install the egg"),
  })
  return (
    <div className="space-y-8">
      <EggHeader
        spec={s}
        actions={
          <>
            <div className="flex items-center gap-2">
              <Checkbox
                id="auto-update"
                checked={autoUpdate}
                disabled={keepsAutoUpdate}
                onCheckedChange={setAutoUpdate}
              />
              <Label htmlFor="auto-update">Update automatically</Label>
            </div>
            {installed && (
              <Link to={`/eggs/${installed.metadata.name}`} className={cn(buttonVariants({ variant: "outline" }))}>
                Open installed egg
              </Link>
            )}
            <Button disabled={install.isPending} onClick={() => install.mutate({ url: egg.url, autoUpdate })}>
              {install.isPending ? <Spinner /> : <DownloadIcon />}
              {installed ? "Install again" : "Install"}
            </Button>
          </>
        }
      >
        <Badge variant="outline" className="font-mono">
          {egg.format}
        </Badge>
        <a
          className="inline-flex items-center gap-1 font-mono text-xs underline-offset-4 hover:underline"
          href={egg.url}
          target="_blank"
          rel="noreferrer"
        >
          {repositoryName(egg.repository)}/{egg.path}
          <ExternalLinkIcon className="size-3" />
        </a>
        {installed && <Badge className="bg-success/15 text-success">Installed</Badge>}
      </EggHeader>
      <p className="text-sm text-muted-foreground">
        {keepsAutoUpdate
          ? "Installing again replaces the installed egg; turn auto update off in the egg editor."
          : installed
            ? "Installing again replaces the installed egg."
            : "The file's URL becomes the egg's update URL."}
      </p>
      <EggTabs spec={s} />
    </div>
  )
}
