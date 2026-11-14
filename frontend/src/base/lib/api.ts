// Client of the base API (backend/base/api; same origin, session cookie): `request` and the calls every app shares.
// The app's lib/api.ts adds its own areas to `baseApi`. Only the data hooks (lib/queries) call it (oxlint);
// components may use `ApiError` (status, field errors).

import type {
  AcceptInviteRequest,
  Branding,
  CreatedInvite,
  CreatedToken,
  CreateInviteRequest,
  CreateUserRequest,
  InviteDetails,
  InviteView,
  LegalTexts,
  LoginResponse,
  OIDCSignIn,
  PanelInfo,
  PanelSettings,
  RequestRates,
  SetupRequest,
  SetupStatus,
  TokenView,
  UpdateSettingsRequest,
  UpdateUserRequest,
  UpgradeJob,
  UpgradeStatus,
  UserView,
  Versions,
} from "@/base/lib/types"

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

/** Sends a request to /api; JSON responses are parsed unless `as` is "text". Errors become ApiError. */
export async function request<T>(
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
      // Not a JSON error body: the status text stays the message.
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

/** A query string from plain values. */
export const q = (params: Record<string, string>) => new URLSearchParams(params).toString()
/** The path of one object below `base`; names are escaped, so they never add path segments. */
export const at = (base: string, name: string) => `${base}/${encodeURIComponent(name)}`
/** The query of reads that ask their source again instead of the panel's kept answer. */
export const refreshed = (refresh: boolean) => (refresh ? "?refresh=true" : "")

/** The calls every app shares, grouped by area like backend/base/api. */
export const baseApi = {
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
    /** Requests per second of the user and, for admins, of the whole panel; polling it does not count. */
    getRequestRates: () => request<RequestRates>("GET", "/request-rates"),
  },
  setup: {
    getStatus: () => request<SetupStatus>("GET", "/setup"),
    run: (body: SetupRequest) => request<LoginResponse>("POST", "/setup", body),
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
    deleteToken: (id: string) => request<void>("DELETE", at("/auth/tokens", id)),
  },
  users: {
    list: () => request<UserView[]>("GET", "/users"),
    create: (body: CreateUserRequest) => request<UserView>("POST", "/users", body),
    update: (name: string, body: UpdateUserRequest) => request<UserView>("PATCH", at("/users", name), body),
    delete: (name: string) => request<void>("DELETE", at("/users", name)),
  },
  invites: {
    list: () => request<InviteView[]>("GET", "/invites"),
    create: (body: CreateInviteRequest) => request<CreatedInvite>("POST", "/invites", body),
    /** A new link, valid for the full lifetime again; the old one stops working. */
    renew: (id: string) => request<CreatedInvite>("POST", `${at("/invites", id)}/renew`),
    delete: (id: string) => request<void>("DELETE", at("/invites", id)),
    /** Whether an invite link works (public). */
    get: (token: string) => request<InviteDetails>("GET", `/auth/invite?${q({ token })}`),
    /** Creates the account of an invite link and signs it in (public). */
    accept: (body: AcceptInviteRequest) => request<LoginResponse>("POST", "/auth/invite", body),
  },
  upgrade: {
    getStatus: (refresh = false) => request<UpgradeStatus>("GET", `/upgrade${refreshed(refresh)}`),
    start: (version: string) => request<UpgradeJob>("POST", "/upgrade", { version }),
  },
  settings: {
    get: () => request<PanelSettings>("GET", "/settings"),
    update: (body: UpdateSettingsRequest) => request<PanelSettings>("PUT", "/settings", body),
  },
}

/** Links the browser opens itself; no request is made here. */
export const baseUrls = {
  /** Sign-in through the identity provider (a browser redirect), returning to next afterwards. */
  oidcStart: (next: string) => `/api/auth/oidc/start?${q({ next })}`,
}
