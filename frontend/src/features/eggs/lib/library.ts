import type { Egg, LibraryEgg } from "@/lib/types"

/** The installed egg a library egg became: same UUID, or imported from its file. */
export const installedAs = (eggs: Egg[] | undefined, e: LibraryEgg) =>
  eggs?.find(
    (i) =>
      (e.uuid !== "" && i.spec.source?.uuid === e.uuid) ||
      i.spec.source?.updateUrl === e.url ||
      i.spec.source?.importedFrom === e.url,
  )

/** Whether every word of the search occurs in one of the texts (case-insensitive). */
export function matches(search: string, ...texts: (string | undefined)[]) {
  const text = texts.join(" ").toLowerCase()
  return search
    .toLowerCase()
    .split(/\s+/)
    .every((word) => text.includes(word))
}

/** The overview page of a library egg. */
export const libraryEggPath = (e: Pick<LibraryEgg, "repository" | "path">) =>
  `/eggs/library/egg?${new URLSearchParams({ repository: e.repository, path: e.path })}`

/** "owner/repo" of a repository URL. */
export const repositoryName = (url: string) => url.replace(/^https:\/\/github\.com\//, "")

/** The folder of an egg file in its repository ("minecraft/java/paper"). */
export const folderOf = (path: string) => path.split("/").slice(0, -1).join("/")
