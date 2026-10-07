import { useQuery, type QueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { CreateServerRequest, GameServer, PowerSignal, UpdateServerRequest } from "@/lib/types"

export const useServers = () =>
  useQuery({
    queryKey: keys.servers,
    queryFn: api.servers.list,
    refetchInterval: 3000,
  })

export const useServer = (server: string) =>
  useQuery({
    queryKey: keys.server(server),
    queryFn: () => api.servers.get(server),
    refetchInterval: 2000,
    enabled: server !== "",
  })

export const useServerStats = (server: string, enabled = true) =>
  useQuery({
    queryKey: keys.serverStats(server),
    queryFn: () => api.servers.getStats(server),
    refetchInterval: 3000,
    enabled,
  })

/** Diagnostics of a server; they make network requests, so they run on demand only (refetch). */
export const useServerDiagnostics = (server: string) =>
  useQuery({
    queryKey: keys.serverDiagnostics(server),
    queryFn: () => api.servers.getDiagnostics(server),
    staleTime: Infinity,
    gcTime: 0,
    refetchOnWindowFocus: false,
  })

/** Reloads the job list of a server: after a job was started or cancelled. */
export const refreshJobs = (server: string) => (qc: QueryClient) => refresh(qc, keys.jobs(server))

/** Background jobs (backup, restore, pull, compress, decompress); polled faster while one runs. */
export const useServerJobs = (server: string) =>
  useQuery({
    queryKey: keys.jobs(server),
    queryFn: () => api.servers.listJobs(server),
    refetchInterval: (q) => (q.state.data?.some((j) => j.state === "running") ? 2000 : 10_000),
  })

/** Stops a running job; the job list shows it as cancelled once its processes have ended. */
export const useCancelJob = (server: string, cb?: MutationCallbacks<void, string>) =>
  useApiMutation((job) => api.servers.cancelJob(server, job), refreshJobs(server), cb)

const storeServer = (qc: QueryClient, gs: GameServer) => {
  qc.setQueryData(keys.server(gs.metadata.name), gs)
  refresh(qc, keys.servers)
}

export const useCreateServer = (cb?: MutationCallbacks<GameServer, CreateServerRequest>) =>
  useApiMutation(api.servers.create, (qc) => refresh(qc, keys.servers), cb)

export const useUpdateServer = (server: string, cb?: MutationCallbacks<GameServer, UpdateServerRequest>) =>
  useApiMutation((changes) => api.servers.update(server, changes), storeServer, cb)

/** Changes the server and starts (or restarts) it, e.g. after an egg feature prompt. */
export const useUpdateAndStartServer = (
  server: string,
  cb?: MutationCallbacks<GameServer, { changes: UpdateServerRequest; signal: "start" | "restart" }>,
) =>
  useApiMutation(
    async ({ changes, signal }) => {
      const gs = await api.servers.update(server, changes)
      await api.servers.sendPower(server, signal)
      return gs
    },
    storeServer,
    cb,
  )

/** Accepts the Minecraft EULA (writes eula.txt) and starts the server. */
export const useAcceptEula = (server: string, cb?: MutationCallbacks<void, void>) =>
  useApiMutation(
    async () => {
      await api.files.write(server, "eula.txt", "eula=true\n")
      await api.servers.sendPower(server, "start")
    },
    (qc) => refresh(qc, keys.server(server)),
    cb,
  )

/** Deletes a server; it leaves the list at once, before the list is loaded again. */
export const useDeleteServer = (server: string, cb?: MutationCallbacks<void, void>) =>
  useApiMutation(
    () => api.servers.delete(server),
    (qc) => {
      void qc.cancelQueries({ queryKey: keys.servers })
      qc.setQueryData<GameServer[]>(keys.servers, (list) => list?.filter((s) => s.metadata.name !== server))
      refresh(qc, keys.servers)
    },
    cb,
  )

/** Sends a power signal (start/stop/restart/kill). */
export const useSendPower = (server: string, cb?: MutationCallbacks<void, PowerSignal>) =>
  useApiMutation(
    (signal) => api.servers.sendPower(server, signal),
    (qc) => refresh(qc, keys.server(server), keys.servers),
    cb,
  )

export const useReinstallServer = (server: string, cb?: MutationCallbacks<void, void>) =>
  useApiMutation(
    () => api.servers.reinstall(server),
    (qc) => refresh(qc, keys.server(server)),
    cb,
  )

/** Moves a stopped server with its data to another user; admins only. */
export const useTransferServer = (server: string, cb?: MutationCallbacks<GameServer, string>) =>
  useApiMutation((owner) => api.servers.transfer(server, owner), storeServer, cb)

/** Suspends (stops and locks for its owner) or unsuspends a server; admins only. */
export const useSuspendServer = (server: string, cb?: MutationCallbacks<GameServer, boolean>) =>
  useApiMutation((suspended) => api.servers.suspend(server, suspended), storeServer, cb)
