import { useEffect, useState } from "react"
import { LibraryIcon, XIcon } from "lucide-react"

import { CheckStatusIcon } from "@/components/common/check-status-icon"
import { EggCountBadge } from "@/components/common/egg-count-badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { useLibrary, useLibraryRepository } from "@/lib/queries"
import type { LibraryRepository } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/** The panel accepts at most this many repositories (settings.maxEggLibraries). */
const maxRepositories = 20

/**
 * The git repositories the egg library lists: one field each, plus an empty one for the next as
 * soon as the last has text. Saved repositories show what the library read (the same query as the
 * Library tab); a new URL is checked while it is typed.
 */
export function EggLibrariesCard({
  repositories,
  onChange,
  error,
}: {
  repositories: string[]
  onChange: (repositories: string[]) => void
  error?: string
}) {
  const library = useLibrary()
  const full = repositories.length >= maxRepositories || repositories.at(-1) === ""
  const rows = full ? repositories : [...repositories, ""]
  // Empty fields at the end are not part of the value (the draft stays clean).
  const change = (next: string[]) => {
    while (next.at(-1) === "") next.pop()
    onChange(next)
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <LibraryIcon className="size-4" />
          Egg library
        </CardTitle>
        <CardDescription>Git repositories whose eggs the library offers to install.</CardDescription>
      </CardHeader>
      <CardContent>
        <Field data-invalid={error ? true : undefined}>
          <FieldLabel htmlFor="egg-library-0">Repositories</FieldLabel>
          {rows.map((url, i) => (
            <RepositoryRow
              key={i}
              id={`egg-library-${i}`}
              url={url}
              listed={library.isPending ? "loading" : library.data?.find((r) => r.url === url.trim())}
              removable={url !== "" || i < rows.length - 1}
              onChange={(v) => change(rows.map((r, j) => (j === i ? v : r)))}
              onRemove={() => change(rows.filter((_, j) => j !== i))}
            />
          ))}
          {error && <FieldError>{error}</FieldError>}
        </Field>
      </CardContent>
    </Card>
  )
}

function RepositoryRow({
  id,
  url,
  listed,
  removable,
  onChange,
  onRemove,
}: {
  id: string
  url: string
  /** The repository as the library read it (saved ones), "loading" while the library loads. */
  listed: LibraryRepository | "loading" | undefined
  removable: boolean
  onChange: (url: string) => void
  onRemove: () => void
}) {
  const typed = url.trim()
  // Checked half a second after the last key press.
  const [checked, setChecked] = useState(typed)
  useEffect(() => {
    const t = setTimeout(() => setChecked(typed), 500)
    return () => clearTimeout(t)
  }, [typed])
  const known = listed === "loading" ? undefined : listed
  const check = useLibraryRepository(listed ? "" : checked)
  const pending = listed === "loading" || (!known && (typed !== checked || check.isFetching))
  const problem = pending
    ? undefined
    : known
      ? known.error
      : check.error
        ? (fieldErrors(check.error).url ?? check.error.message)
        : check.data?.error
  const eggs = (known ? known.eggs.length : check.data?.eggs) ?? 0

  return (
    <div className="grid gap-1">
      <div className="flex items-center gap-2">
        <Input
          id={id}
          className="font-mono text-sm"
          placeholder="https://<host>/<owner>/<repo>"
          value={url}
          onChange={(e) => onChange(e.target.value)}
          aria-invalid={problem ? true : undefined}
        />
        <div className="flex w-20 shrink-0 items-center gap-1.5">
          {!typed ? null : pending ? (
            <Spinner />
          ) : problem ? (
            <CheckStatusIcon status="error" className="size-4" />
          ) : (
            <>
              <CheckStatusIcon status={eggs ? "ok" : "warning"} className="size-4" />
              <EggCountBadge count={eggs} />
            </>
          )}
        </div>
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          className={removable ? undefined : "invisible"}
          onClick={onRemove}
          aria-label="Remove repository"
        >
          <XIcon />
        </Button>
      </div>
      {problem && <p className="text-xs text-destructive">{problem}</p>}
    </div>
  )
}
