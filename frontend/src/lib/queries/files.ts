import { useState } from "react"
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import { refreshJobs } from "@/lib/queries/servers"
import type { FilesSession, ServerJob } from "@/lib/types"

type FilesView = "ready" | "starting" | "stopped"

/**
 * Starts the file container of a server when the file manager opens and follows its
 * state. The container is removed after a minute without file operations; the view then
 * turns "stopped" until the user starts it again.
 */
export function useFilesSession(server: string) {
  // Whether the container was ready during this visit (distinguishes "not started yet"
  // from "stopped after inactivity").
  const [wasReady, setWasReady] = useState(false)
  const qc = useQueryClient()
  const track = (s: FilesSession) => {
    if (s.ready) setWasReady(true)
    // The light on the Files tab follows at once (see useFilesPodState).
    qc.setQueryData(keys.filesPodState(server), s)
    return s
  }
  const open = useQuery({
    queryKey: keys.filesOpen(server),
    queryFn: () => api.files.openSession(server).then(track),
    staleTime: Infinity,
    gcTime: 0,
    refetchOnWindowFocus: false,
  })
  // Polling the state does not count as activity, so an open tab does not keep it alive.
  // Until the container was ready once, the poll keeps asking for it: the start can take
  // over a minute and a pod that vanished meanwhile is created again.
  const state = useQuery({
    queryKey: keys.filesSession(server),
    queryFn: () => (wasReady ? api.files.getSession(server) : api.files.openSession(server)).then(track),
    enabled: open.isSuccess,
    // Ready: every 5 s, so the countdown of the file page follows file operations quickly.
    refetchInterval: (q) => (q.state.data?.ready ? 5_000 : 1_500),
  })
  const session = state.data ?? open.data
  const view: FilesView = session?.ready ? "ready" : wasReady && session?.state !== "Starting" ? "stopped" : "starting"

  return {
    session,
    // When the session answer arrived (its stopsInSeconds count from then).
    updatedAt: state.data ? state.dataUpdatedAt : open.dataUpdatedAt,
    view,
    error: open.error,
    restart: () => {
      setWasReady(false)
      void open.refetch().then(() => state.refetch())
    },
  }
}

/**
 * State of the file container for the light on the Files tab. Only reads it (GET): watching
 * neither starts the container nor keeps it running; useFilesSession writes its answers here.
 */
export const useFilesPodState = (server: string) =>
  useQuery({
    queryKey: keys.filesPodState(server),
    queryFn: () => api.files.getSession(server),
    refetchInterval: (q) => (q.state.data?.state === "Starting" ? 2_000 : 10_000),
  })

/** Entries of a directory; needs the file container. */
export const useFileList = (server: string, dir: string, enabled: boolean) =>
  useQuery({
    queryKey: keys.files(server, dir),
    queryFn: () => api.files.list(server, dir),
    enabled,
  })

/** Contents of a file for the editor; not kept after the editor closes. */
export const useFileContent = (server: string, file: string, enabled: boolean) =>
  useQuery({
    queryKey: keys.fileContent(server, file),
    queryFn: () => api.files.read(server, file),
    enabled,
    gcTime: 0,
  })

/** Reloads listings and backups, e.g. after a background job changed the files. */
export function useRefreshFiles(server: string) {
  const qc = useQueryClient()
  return () => refresh(qc, keys.allFiles(server), keys.backups(server))
}

// Every file operation changes listings.
const refreshListings = (server: string) => (qc: QueryClient) => refresh(qc, keys.allFiles(server))

export const useWriteFile = (server: string, cb?: MutationCallbacks<void, { file: string; content: string }>) =>
  useApiMutation(({ file, content }) => api.files.write(server, file, content), refreshListings(server), cb)

export const useUploadFiles = (server: string, cb?: MutationCallbacks<void, { dir: string; files: File[] }>) =>
  useApiMutation(({ dir, files }) => api.files.upload(server, dir, files), refreshListings(server), cb)

export const useCreateFolder = (server: string, cb?: MutationCallbacks<void, { dir: string; name: string }>) =>
  useApiMutation(({ dir, name }) => api.files.createFolder(server, dir, name), refreshListings(server), cb)

/** Renames an entry of dir; a name with "/" moves it into another folder. */
export const useRenameFile = (
  server: string,
  cb?: MutationCallbacks<void, { dir: string; from: string; to: string }>,
) => useApiMutation(({ dir, from, to }) => api.files.rename(server, dir, from, to), refreshListings(server), cb)

/** Renames (moves) entries by full paths from → to, one after the other. */
export const useRenameFiles = (server: string, cb?: MutationCallbacks<void, { from: string; to: string }[]>) =>
  useApiMutation(
    async (moves) => {
      for (const m of moves) await api.files.rename(server, "/", m.from, m.to)
    },
    refreshListings(server),
    cb,
  )

/** Starts a job that packs entries of dir into a new archive; the job list shows its progress. */
export const useCompressFiles = (server: string, cb?: MutationCallbacks<ServerJob, { dir: string; names: string[] }>) =>
  useApiMutation(({ dir, names }) => api.files.compress(server, dir, names), refreshJobs(server), cb)

/** Starts a job that extracts an archive into its folder; the job list shows its progress. */
export const useDecompressFile = (server: string, cb?: MutationCallbacks<ServerJob, { dir: string; name: string }>) =>
  useApiMutation(({ dir, name }) => api.files.decompress(server, dir, name), refreshJobs(server), cb)

export const useDeleteFiles = (server: string, cb?: MutationCallbacks<void, { dir: string; names: string[] }>) =>
  useApiMutation(({ dir, names }) => api.files.delete(server, dir, names), refreshListings(server), cb)

/** Starts a download job from a URL into dir; the job list shows its progress. */
export const usePullFile = (
  server: string,
  cb?: MutationCallbacks<ServerJob, { url: string; dir: string; filename?: string }>,
) => useApiMutation(({ url, dir, filename }) => api.files.pull(server, url, dir, filename), refreshJobs(server), cb)
