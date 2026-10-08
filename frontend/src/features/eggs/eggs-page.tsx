import { useState } from "react"
import { EggIcon as EggLucide, PlusIcon, SearchIcon } from "lucide-react"
import { Link, useSearchParams } from "react-router"

import { EggCountBadge } from "@/components/common/egg-count-badge"
import { EmptyState, QueryState, SkeletonGrid } from "@/components/common/query-state"
import { ScrollableTabsList } from "@/components/common/scrollable-tabs-list"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Tabs, TabsContent, TabsTrigger } from "@/components/ui/tabs"
import { EggCard } from "@/features/eggs/components/egg-card"
import { EggImportDialog } from "@/features/eggs/components/egg-import-dialog"
import { EggLibrary } from "@/features/eggs/components/egg-library"
import { matches } from "@/features/eggs/lib/library"
import { useEggs, useLibrary, useServers } from "@/lib/queries"

/**
 * /eggs (admins): the installed eggs (import from a file or URL, or create one with the wizard)
 * and the library: the eggs of git repositories, ready to install (?tab=library).
 */
export function EggsPage() {
  const [params, setParams] = useSearchParams()
  const tab = params.get("tab") === "library" ? "library" : "installed"
  const [search, setSearch] = useState("")
  const installed = useEggs().data
  const library = useLibrary().data
  return (
    <div className="space-y-8">
      <PageHeader
        title="Eggs"
        description="Server templates from Pterodactyl (PTDL_v1/v2) and Pelican (PLCN_v1-v3)."
        actions={
          <>
            <Button variant="outline" asChild>
              <Link to="/eggs/new">
                <PlusIcon />
                New egg
              </Link>
            </Button>
            <EggImportDialog />
          </>
        }
      />
      <Tabs value={tab} onValueChange={(v) => setParams(v === "library" ? { tab: v } : {}, { replace: true })}>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <ScrollableTabsList>
            <TabsTrigger value="installed">
              Installed
              {installed && <EggCountBadge count={installed.length} />}
            </TabsTrigger>
            <TabsTrigger value="library">
              Library
              {library && <EggCountBadge count={library.reduce((n, r) => n + r.eggs.length, 0)} />}
            </TabsTrigger>
          </ScrollableTabsList>
          <InputGroup className="sm:max-w-xs">
            <InputGroupAddon>
              <SearchIcon />
            </InputGroupAddon>
            <InputGroupInput
              placeholder="Search name, author, tag…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </InputGroup>
        </div>
        <TabsContent value="installed" className="pt-4">
          <InstalledEggs search={search} />
        </TabsContent>
        <TabsContent value="library" className="pt-4">
          <EggLibrary search={search} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function InstalledEggs({ search }: { search: string }) {
  const eggs = useEggs()
  const servers = useServers()
  const usage = new Map<string, number>()
  servers.data?.forEach((s) => usage.set(s.spec.eggRef, (usage.get(s.spec.eggRef) ?? 0) + 1))
  return (
    <QueryState
      query={eggs}
      skeleton={<SkeletonGrid />}
      empty={
        <EmptyState
          icon={<EggLucide />}
          title="No eggs installed"
          description="Install one from the library, import a file or URL, or create your own."
        >
          <Button variant="outline" asChild>
            <Link to="/eggs?tab=library">Open the library</Link>
          </Button>
          <EggImportDialog />
        </EmptyState>
      }
    >
      {(list) => {
        const shown = list.filter((e) =>
          matches(search, e.spec.displayName, e.spec.author, e.spec.description, ...(e.spec.tags ?? [])),
        )
        return shown.length === 0 ? (
          <p className="py-10 text-center text-sm text-muted-foreground">No egg matches the search.</p>
        ) : (
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
            {shown.map((egg) => (
              <EggCard key={egg.metadata.uid} egg={egg} servers={usage.get(egg.metadata.name) ?? 0} />
            ))}
          </div>
        )
      }}
    </QueryState>
  )
}
