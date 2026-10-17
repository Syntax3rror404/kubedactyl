import { useQuery, type QueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, refreshes, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { Egg, EggExportFormat, EggSpec, LibraryRepository } from "@/lib/types"

/** All eggs. */
export const useEggs = () => useQuery({ queryKey: keys.eggs, queryFn: api.eggs.list })

/** One egg; nothing is loaded without a name. */
export const useEgg = (name: string) =>
  useQuery({ queryKey: keys.egg(name), queryFn: () => api.eggs.get(name), enabled: name !== "" })

/** The export of an egg as text (preview in the export dialog); always fresh. */
export const useEggExport = (name: string, format: EggExportFormat, enabled: boolean) =>
  useQuery({
    queryKey: keys.eggExport(name, format),
    queryFn: () => api.eggs.export(name, format),
    enabled,
    staleTime: 0,
    gcTime: 0,
  })

// Stored from the answer: a list loaded right away could come from the panel's lagging informer cache.
const storeEgg = (qc: QueryClient, egg: Egg) => {
  qc.setQueryData(keys.egg(egg.metadata.name), egg)
  refresh(qc, keys.eggs)
}

/** Imports an egg from a file or a URL (optionally with the hourly auto update turned on). */
export const useImportEgg = (cb?: MutationCallbacks<Egg, { file: File } | { url: string; autoUpdate?: boolean }>) =>
  useApiMutation(
    (src) => ("file" in src ? api.eggs.importFile(src.file) : api.eggs.importUrl(src.url, src.autoUpdate)),
    storeEgg,
    cb,
  )

/** The eggs of the egg library; the panel keeps the repositories for 15 minutes, so does the browser. */
export const useLibrary = () =>
  useQuery({ queryKey: keys.library, queryFn: () => api.eggs.listLibrary(), staleTime: 15 * 60_000 })

/** Downloads the library's repositories again. */
export const useRefreshLibrary = (cb?: MutationCallbacks<LibraryRepository[], void>) =>
  useApiMutation(
    () => api.eggs.listLibrary(true),
    (qc, repos) => qc.setQueryData(keys.library, repos),
    cb,
  )

/** Checks one repository, also one not saved yet (the settings check a URL while it is typed). */
export const useLibraryRepository = (url: string) =>
  useQuery({
    queryKey: keys.libraryRepository(url),
    queryFn: () => api.eggs.getLibraryRepository(url),
    enabled: url !== "",
    staleTime: 15 * 60_000,
    retry: false,
  })

/** One egg of the library with its content (downloaded when opened). */
export const useLibraryEgg = (repository: string, path: string) =>
  useQuery({
    queryKey: keys.libraryEgg(repository, path),
    queryFn: () => api.eggs.getLibraryEgg(repository, path),
    enabled: repository !== "" && path !== "",
  })

/** Creates an egg made in the editor. */
export const useCreateEgg = (cb?: MutationCallbacks<Egg, EggSpec>) => useApiMutation(api.eggs.create, storeEgg, cb)

/** Saves an egg edited in the editor. */
export const useUpdateEgg = (name: string, cb?: MutationCallbacks<Egg, EggSpec>) =>
  useApiMutation((spec) => api.eggs.update(name, spec), storeEgg, cb)

/** Replaces an egg with the file at its update URL. */
export const useUpdateEggFromUrl = (name: string, cb?: MutationCallbacks<Egg, void>) =>
  useApiMutation(() => api.eggs.updateFromUrl(name), storeEgg, cb)

/** Deletes an egg that no server uses. */
export const useDeleteEgg = (name: string, cb?: MutationCallbacks<void, void>) =>
  useApiMutation(() => api.eggs.delete(name), refreshes(keys.eggs), cb)
