/**
 * React Query cache keys of the base; the app's lib/queries/keys.ts adds its own. refresh() matches a key exactly,
 * so reloading a list never reloads the details below it; refreshAll() reloads a group.
 */
export const baseKeys = {
  me: ["me"] as const,
  setup: ["setup"] as const,
  oidcSignIn: ["oidc-sign-in"] as const,
  legal: ["legal"] as const,
  branding: ["branding"] as const,
  licenses: ["licenses"] as const,
  apiDocument: ["api-document"] as const,
  info: ["info"] as const,
  liveInfo: ["info", "live"] as const,
  requestRates: ["request-rates"] as const,
  tokens: ["me", "tokens"] as const,
  users: ["users"] as const,
  invites: ["invites"] as const,
  invite: (token: string) => ["invite", token] as const,
  settings: ["settings"] as const,
  upgrade: ["upgrade"] as const,
  versions: ["versions"] as const,
}
