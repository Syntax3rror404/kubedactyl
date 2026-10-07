// Typed client for the Kubedactyl REST API (same origin, session cookie).
//
// Only the data hooks in lib/queries call `api` (enforced by oxlint); components use those hooks.
// Components may use `ApiError` (status, field errors) and `urls` (links for downloads).

import type {
  AcceptInviteRequest,
  BackupList,
  Branding,
  ClusterHealth,
  ClusterIdentity,
  ClusterInfo,
  CreatedInvite,
  CreatedToken,
  CreateInviteRequest,
  CreateServerRequest,
  CreateUserRequest,
  DirectoryListing,
  Egg,
  EggExportFormat,
  EggList,
  EggSpec,
  FilesSession,
  GameServer,
  GameServerList,
  InviteDetails,
  InviteView,
  JobList,
  LegalTexts,
  LibraryEggContent,
  LibraryList,
  LoginResponse,
  NodesResponse,
  OIDCSignIn,
  PanelInfo,
  PanelSettings,
  PoolList,
  PowerSignal,
  RequestRates,
  Schedule,
  ScheduleList,
  ServerDiagnostics,
  ServerJob,
  ServerStats,
  SetupStatus,
  StorageClassList,
  TokenView,
  UpdateServerRequest,
  UpdateSettingsRequest,
  UpdateUserRequest,
  UpgradeJob,
  UpgradeStatus,
  UserView,
  Versions,
} from "@/lib/types"

/** ApiError carries the HTTP status and per-field validation messages. */
export class ApiError extends Error {
  status: number
  fields?: Record<string, string>

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message)
    this.status = status
    this.fields = fields
  }
}

let onUnauthorized: (() => void) | null = null

/** Registers what happens when the session is gone (the auth provider redirects to /login). */
export function setUnauthorizedHandler(fn: (() => void) | null) {
  onUnauthorized = fn
}

/** A refused request: too many from this user (429) or the panel is busy (503); try again after `seconds`. */
export type Throttle = { busy: boolean; seconds: number }

let onThrottled: ((t: Throttle) => void) | null = null

/** Registers what happens when the panel refuses a request for now (a notice until Retry-After). */
export function setThrottledHandler(fn: ((t: Throttle) => void) | null) {
  onThrottled = fn
}

/** Sends a request; JSON responses are parsed unless `as` is "text". */
async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  init?: RequestInit,
  as: "auto" | "text" = "auto",
): Promise<T> {
  const headers: Record<string, string> = {}
  let payload: BodyInit | undefined
  if (body instanceof FormData || typeof body === "string" || body instanceof Blob) {
    payload = body
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json"
    payload = JSON.stringify(body)
  }
  const res = await fetch(`/api${path}`, { method, headers, body: payload, ...init })
  if (res.status === 401 && !path.startsWith("/auth/login")) onUnauthorized?.()
  const retryAfter = Number(res.headers.get("retry-after"))
  if ((res.status === 429 || res.status === 503) && retryAfter > 0)
    onThrottled?.({ busy: res.status === 503, seconds: retryAfter })
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    let fields: Record<string, string> | undefined
    try {
      const data = await res.json()
      message = data.error ?? message
      fields = data.fields
    } catch {
      // not a JSON error body
    }
    throw new ApiError(res.status, message, fields)
  }
  if (res.status === 204) return undefined as T
  const type = res.headers.get("content-type") ?? ""
  return (as === "auto" && type.includes("application/json") ? res.json() : res.text()) as Promise<T>
}

/** Reads a file of the production build; the SPA fallback (index.html) means the build has none. */
async function buildFile(path: string): Promise<string> {
  const res = await fetch(path)
  if (!res.ok || res.headers.get("content-type")?.includes("text/html"))
    throw new ApiError(404, "Written by the production build (make build); not there in development mode.")
  return res.text()
}

const q = (params: Record<string, string>) => new URLSearchParams(params).toString()
const srv = (name: string) => `/servers/${encodeURIComponent(name)}`
const egg = (name: string) => `/eggs/${encodeURIComponent(name)}`

/** All API calls, grouped by area. Errors become ApiError (status + per-field messages). */
export const api = {
  panel: {
    /** Name, version and mode of the panel (public). */
    getInfo: (init?: RequestInit) => request<PanelInfo>("GET", "/info", undefined, init),
    /** Imprint and privacy policy (public). */
    getLegalTexts: () => request<LegalTexts>("GET", "/legal"),
    getBranding: () => request<Branding>("GET", "/branding"),
    /** License texts of the panel and its dependencies as Markdown (public, production build only). */
    getLicenses: () => buildFile("/third-party-licenses.md"),
    /** Versions of the panel, Go and the main Go modules (admins). */
    getVersions: () => request<Versions>("GET", "/versions"),
    /** Requests per second of the user and, for admins, Kubernetes API calls of the panel (polling does not count). */
    getRequestRates: () => request<RequestRates>("GET", "/request-rates"),
  },
  setup: {
    getStatus: () => request<SetupStatus>("GET", "/setup"),
    run: (body: { username: string; password: string; displayName?: string; token: string }) =>
      request<LoginResponse>("POST", "/setup", body),
  },
  auth: {
    login: (username: string, password: string) =>
      request<LoginResponse>("POST", "/auth/login", { username, password }),
    logout: () => request<void>("POST", "/auth/logout"),
    logoutAll: () => request<void>("POST", "/auth/logout-all"),
    getMe: () => request<UserView>("GET", "/auth/me"),
    getOIDCSignIn: () => request<OIDCSignIn>("GET", "/auth/oidc"),
    updatePassword: (current: string, next: string, revokeTokens: boolean) =>
      request<void>("PUT", "/auth/password", { current, new: next, revokeTokens }),
    listTokens: () => request<TokenView[]>("GET", "/auth/tokens"),
    createToken: (name: string, expiresInDays: number) =>
      request<CreatedToken>("POST", "/auth/tokens", { name, expiresInDays }),
    deleteToken: (id: string) => request<void>("DELETE", `/auth/tokens/${encodeURIComponent(id)}`),
  },
  users: {
    list: () => request<UserView[]>("GET", "/users"),
    create: (body: CreateUserRequest) => request<UserView>("POST", "/users", body),
    update: (name: string, body: UpdateUserRequest) =>
      request<UserView>("PATCH", `/users/${encodeURIComponent(name)}`, body),
    delete: (name: string) => request<void>("DELETE", `/users/${encodeURIComponent(name)}`),
  },
  invites: {
    list: () => request<InviteView[]>("GET", "/invites"),
    create: (body: CreateInviteRequest) => request<CreatedInvite>("POST", "/invites", body),
    /** A new link, valid for the full lifetime again; the old one stops working. */
    renew: (id: string) => request<CreatedInvite>("POST", `/invites/${encodeURIComponent(id)}/renew`),
    delete: (id: string) => request<void>("DELETE", `/invites/${encodeURIComponent(id)}`),
    /** Whether an invite link works (public). */
    get: (token: string) => request<InviteDetails>("GET", `/auth/invite?${q({ token })}`),
    /** Creates the account of an invite link and signs it in (public). */
    accept: (body: AcceptInviteRequest) => request<LoginResponse>("POST", "/auth/invite", body),
  },
  upgrade: {
    getStatus: (refresh = false) => request<UpgradeStatus>("GET", `/upgrade${refresh ? "?refresh=true" : ""}`),
    start: (version: string) => request<UpgradeJob>("POST", "/upgrade", { version }),
  },
  cluster: {
    getInfo: () => request<ClusterInfo>("GET", "/cluster"),
    listNodes: () => request<NodesResponse>("GET", "/cluster/nodes"),
    probeNodes: () => request<NodesResponse>("POST", "/cluster/nodes/probe"),
    getIdentity: () => request<ClusterIdentity>("GET", "/cluster/identity"),
    getHealth: () => request<ClusterHealth>("GET", "/cluster/health"),
  },
  settings: {
    get: () => request<PanelSettings>("GET", "/settings"),
    update: (body: UpdateSettingsRequest) => request<PanelSettings>("PUT", "/settings", body),
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
      request<LibraryList>("GET", `/egg-library${refresh ? "?refresh=true" : ""}`).then((r) => r.repositories),
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
    sendCommand: (server: string, command: string) => request<void>("POST", `${srv(server)}/command`, { command }),
    reinstall: (server: string) => request<void>("POST", `${srv(server)}/reinstall`),
    suspend: (server: string, suspended: boolean) =>
      request<GameServer>("POST", `${srv(server)}/suspend`, { suspended }),
    transfer: (server: string, owner: string) => request<GameServer>("POST", `${srv(server)}/transfer`, { owner }),
    getStats: (server: string) => request<ServerStats>("GET", `${srv(server)}/stats`),
    getDiagnostics: (server: string) => request<ServerDiagnostics>("GET", `${srv(server)}/diagnostics`),
    listJobs: (server: string) => request<JobList>("GET", `${srv(server)}/jobs`).then((r) => r.items),
    cancelJob: (server: string, job: string) =>
      request<void>("POST", `${srv(server)}/jobs/${encodeURIComponent(job)}/cancel`),
  },
  backups: {
    list: (server: string) => request<BackupList>("GET", `${srv(server)}/backups`).then((r) => r.items),
    create: (server: string, label: string) => request<ServerJob>("POST", `${srv(server)}/backups`, { label }),
    restore: (server: string, backup: string) =>
      request<ServerJob>("POST", `${srv(server)}/backups/${encodeURIComponent(backup)}/restore`),
    delete: (server: string, backup: string) =>
      request<void>("DELETE", `${srv(server)}/backups/${encodeURIComponent(backup)}`),
  },
  schedules: {
    list: (server: string) => request<ScheduleList>("GET", `${srv(server)}/schedules`),
    update: (server: string, items: Schedule[]) => request<ScheduleList>("PUT", `${srv(server)}/schedules`, { items }),
    run: (server: string, schedule: string) =>
      request<void>("POST", `${srv(server)}/schedules/${encodeURIComponent(schedule)}/run`),
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

/** Links the browser opens itself (downloads); no request is made here. */
export const urls = {
  /** Sign-in through the identity provider (a browser redirect), returning to next afterwards. */
  oidcStart: (next: string) => `/api/auth/oidc/start?${q({ next })}`,
  fileDownload: (server: string, file: string) => `/api${srv(server)}/files/download?${q({ file })}`,
  eggExport: (name: string, format: EggExportFormat) => `/api${egg(name)}/export?${q({ format, download: "true" })}`,
}
