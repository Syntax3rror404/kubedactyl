// Client of the Kubedactyl API: the base calls (base/lib/api.ts) and this app's areas. Only lib/queries calls `api`
// (oxlint); components may use `urls` (links the browser opens itself).

import { at, baseApi, baseUrls, q, refreshed, request } from "@/base/lib/api"
import type {
  BackupList,
  ClusterHealth,
  ClusterIdentity,
  ClusterInfo,
  CreateServerRequest,
  DirectoryListing,
  Egg,
  EggExportFormat,
  EggList,
  EggSpec,
  FilesSession,
  GameServer,
  GameServerList,
  JobList,
  LibraryEggContent,
  LibraryList,
  LibraryRepositoryCheck,
  MigrateRequest,
  NodesResponse,
  PoolList,
  PowerSignal,
  Schedule,
  ScheduleList,
  ServerDiagnostics,
  ServerJob,
  ServerStats,
  StorageClassList,
  UpdateServerRequest,
} from "@/lib/types"

const srv = (name: string) => at("/servers", name)
const egg = (name: string) => at("/eggs", name)

/** All API calls: the base ones and this app's, grouped by area. Errors become ApiError (status + per-field messages). */
export const api = {
  ...baseApi,
  cluster: {
    getInfo: () => request<ClusterInfo>("GET", "/cluster"),
    listNodes: () => request<NodesResponse>("GET", "/cluster/nodes"),
    probeNodes: () => request<NodesResponse>("POST", "/cluster/nodes/probe"),
    getIdentity: () => request<ClusterIdentity>("GET", "/cluster/identity"),
    getHealth: () => request<ClusterHealth>("GET", "/cluster/health"),
  },
  settings: {
    ...baseApi.settings,
    listStorageClasses: () => request<StorageClassList>("GET", "/settings/storage-classes").then((r) => r.items),
    listPools: () => request<PoolList>("GET", "/settings/load-balancer-pools"),
  },
  eggs: {
    list: () => request<EggList>("GET", "/eggs").then((r) => r.items),
    get: (name: string) => request<Egg>("GET", egg(name)),
    importFile: (file: File) => {
      const form = new FormData()
      form.append("file", file)
      return request<Egg>("POST", "/eggs/import", form)
    },
    importUrl: (url: string, autoUpdate = false) => request<Egg>("POST", "/eggs/import-url", { url, autoUpdate }),
    /** The eggs of the repositories in the settings; refresh downloads them again. */
    listLibrary: (refresh = false) =>
      request<LibraryList>("GET", `/egg-library${refreshed(refresh)}`).then((r) => r.repositories),
    /** Whether a repository can be read and how many eggs it holds (also one not saved yet). */
    getLibraryRepository: (url: string) =>
      request<LibraryRepositoryCheck>("GET", `/egg-library/repository?${q({ url })}`),
    getLibraryEgg: (repository: string, path: string) =>
      request<LibraryEggContent>("GET", `/egg-library/egg?${q({ repository, path })}`),
    create: (spec: EggSpec) => request<Egg>("POST", "/eggs", spec),
    update: (name: string, spec: EggSpec) => request<Egg>("PUT", egg(name), spec),
    updateFromUrl: (name: string) => request<Egg>("POST", `${egg(name)}/update-from-url`),
    delete: (name: string) => request<void>("DELETE", egg(name)),
    /** The export as text (preview). */
    export: (name: string, format: EggExportFormat) =>
      request<string>("GET", `${egg(name)}/export?${q({ format })}`, undefined, undefined, "text"),
  },
  servers: {
    list: () => request<GameServerList>("GET", "/servers").then((r) => r.items),
    get: (server: string) => request<GameServer>("GET", srv(server)),
    create: (body: CreateServerRequest) => request<GameServer>("POST", "/servers", body),
    update: (server: string, body: UpdateServerRequest) => request<GameServer>("PATCH", srv(server), body),
    delete: (server: string) => request<void>("DELETE", srv(server)),
    sendPower: (server: string, signal: PowerSignal) => request<void>("POST", `${srv(server)}/power`, { signal }),
    reinstall: (server: string) => request<void>("POST", `${srv(server)}/reinstall`),
    suspend: (server: string, suspended: boolean) =>
      request<GameServer>("POST", `${srv(server)}/suspend`, { suspended }),
    transfer: (server: string, owner: string) => request<GameServer>("POST", `${srv(server)}/transfer`, { owner }),
    migrate: (server: string, body: MigrateRequest) => request<GameServer>("POST", `${srv(server)}/migrate`, body),
    cancelMigration: (server: string) => request<GameServer>("POST", `${srv(server)}/migrate/cancel`),
    getStats: (server: string) => request<ServerStats>("GET", `${srv(server)}/stats`),
    getDiagnostics: (server: string) => request<ServerDiagnostics>("GET", `${srv(server)}/diagnostics`),
    listJobs: (server: string) => request<JobList>("GET", `${srv(server)}/jobs`).then((r) => r.items),
    cancelJob: (server: string, job: string) => request<void>("POST", `${at(`${srv(server)}/jobs`, job)}/cancel`),
  },
  backups: {
    list: (server: string) => request<BackupList>("GET", `${srv(server)}/backups`).then((r) => r.items),
    create: (server: string, label: string) => request<ServerJob>("POST", `${srv(server)}/backups`, { label }),
    restore: (server: string, backup: string) =>
      request<ServerJob>("POST", `${at(`${srv(server)}/backups`, backup)}/restore`),
    delete: (server: string, backup: string) => request<void>("DELETE", at(`${srv(server)}/backups`, backup)),
  },
  schedules: {
    list: (server: string) => request<ScheduleList>("GET", `${srv(server)}/schedules`),
    update: (server: string, items: Schedule[]) => request<ScheduleList>("PUT", `${srv(server)}/schedules`, { items }),
    run: (server: string, schedule: string) => request<void>("POST", `${at(`${srv(server)}/schedules`, schedule)}/run`),
  },
  files: {
    /** State of the file container (does not start it). */
    getSession: (server: string) => request<FilesSession>("GET", `${srv(server)}/files/session`),
    /** Starts the file container if needed. */
    openSession: (server: string) => request<FilesSession>("POST", `${srv(server)}/files/session`),
    pull: (server: string, url: string, directory: string, filename?: string) =>
      request<ServerJob>("POST", `${srv(server)}/files/pull`, { url, directory, filename: filename || undefined }),
    list: (server: string, directory: string) =>
      request<DirectoryListing>("GET", `${srv(server)}/files/list?${q({ directory })}`),
    read: (server: string, file: string) => request<string>("GET", `${srv(server)}/files/contents?${q({ file })}`),
    write: (server: string, file: string, content: string) =>
      request<void>("POST", `${srv(server)}/files/write?${q({ file })}`, content),
    createFolder: (server: string, root: string, folder: string) =>
      request<void>("POST", `${srv(server)}/files/create-folder`, { root, name: folder }),
    delete: (server: string, root: string, files: string[]) =>
      request<void>("POST", `${srv(server)}/files/delete`, { root, files }),
    rename: (server: string, root: string, from: string, to: string) =>
      request<void>("PUT", `${srv(server)}/files/rename`, { root, from, to }),
    compress: (server: string, root: string, files: string[]) =>
      request<ServerJob>("POST", `${srv(server)}/files/compress`, { root, files }),
    decompress: (server: string, root: string, file: string) =>
      request<ServerJob>("POST", `${srv(server)}/files/decompress`, { root, file }),
    upload: (server: string, directory: string, files: File[]) => {
      const form = new FormData()
      files.forEach((f) => form.append("files", f, f.name))
      return request<void>("POST", `${srv(server)}/files/upload?${q({ directory })}`, form)
    },
  },
}

/** Links the browser opens itself (downloads, the console websocket); no request is made here. */
export const urls = {
  ...baseUrls,
  /** The console websocket of a server (wss: on HTTPS pages). */
  console: (server: string) =>
    `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/api${srv(server)}/ws`,
  fileDownload: (server: string, file: string) => `/api${srv(server)}/files/download?${q({ file })}`,
  eggExport: (name: string, format: EggExportFormat) => `/api${egg(name)}/export?${q({ format, download: "true" })}`,
}
