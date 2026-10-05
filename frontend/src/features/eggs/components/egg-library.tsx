import { LibraryIcon, RefreshCwIcon, TriangleAlertIcon } from "lucide-react"
import { Link } from "react-router"

import { Callout } from "@/components/common/callout"
import { EmptyState, QueryState, SkeletonGrid } from "@/components/common/query-state"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { LibraryEggCard } from "@/features/eggs/components/library-egg-card"
import { installedAs, matches, repositoryName } from "@/features/eggs/lib/library"
import { formatRelativeTime, plural } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useEggs, useLibrary, useRefreshLibrary } from "@/lib/queries"
import type { Egg, LibraryRepository } from "@/lib/types"

/** The "Library" tab of the eggs page: the eggs of the GitHub repositories in the settings. */
export function EggLibrary({ search }: { search: string }) {
  const library = useLibrary()
  const eggs = useEggs()
  const refresh = useRefreshLibrary({ onError: failed("refresh the library") })
  return (
    <QueryState
      query={library}
      skeleton={<SkeletonGrid />}
      empty={
        <EmptyState icon={<LibraryIcon />} title="No repositories" description="Add egg libraries to list here.">
          <Button variant="outline" asChild>
            <Link to="/settings">Open settings</Link>
          </Button>
        </EmptyState>
      }
    >
      {(repos) => (
        <div className="space-y-8">
          <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-muted-foreground">
            <span>
              {plural(
                repos.reduce((n, r) => n + r.eggs.length, 0),
                "egg",
              )}{" "}
              · loaded {formatRelativeTime(repos[0].fetchedAt)}
            </span>
            <Button variant="outline" size="sm" disabled={refresh.isPending} onClick={() => refresh.mutate()}>
              {refresh.isPending ? <Spinner /> : <RefreshCwIcon />}
              Refresh
            </Button>
          </div>
          {repos.map((r) => (
            <RepositorySection key={r.url} repo={r} installed={eggs.data} search={search} />
          ))}
        </div>
      )}
    </QueryState>
  )
}

function RepositorySection({
  repo,
  installed,
  search,
}: {
  repo: LibraryRepository
  installed: Egg[] | undefined
  search: string
}) {
  const shown = repo.eggs.filter((e) => matches(search, e.name, e.description, e.author, e.path, ...e.tags))
  return (
    <section className="space-y-4">
      <h2 className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <a
          className="font-semibold tracking-tight underline-offset-4 hover:underline"
          href={repo.url}
          target="_blank"
          rel="noreferrer"
        >
          {repositoryName(repo.url)}
        </a>
        <span className="text-sm text-muted-foreground">{plural(repo.eggs.length, "egg")}</span>
      </h2>
      {repo.error ? (
        <Callout tone="danger" icon={<TriangleAlertIcon />} title="The repository could not be read">
          {repo.error}
        </Callout>
      ) : shown.length === 0 ? (
        <p className="text-sm text-muted-foreground">No egg matches the search.</p>
      ) : (
        <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
          {shown.map((e) => (
            <LibraryEggCard key={e.path} egg={e} installed={!!installedAs(installed, e)} />
          ))}
        </div>
      )}
    </section>
  )
}
