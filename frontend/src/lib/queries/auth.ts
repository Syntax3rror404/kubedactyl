import { useEffect } from "react"
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query"

import { api, setUnauthorizedHandler } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { CreatedToken, LoginResponse, TokenView } from "@/lib/types"

/** Whether the panel still needs its first administrator (public endpoint). */
export const useSetupStatus = (enabled = true) =>
  useQuery({ queryKey: keys.setup, queryFn: api.setup.getStatus, enabled, staleTime: 10_000 })

/** Whether the sign-in page offers single sign-on; public. Saving the settings refreshes it. */
export const useOIDCSignIn = () =>
  useQuery({ queryKey: keys.oidcSignIn, queryFn: api.auth.getOIDCSignIn, staleTime: Infinity })

/** The signed-in user; a failed request is not retried (no session: sign in). */
export const useMe = () => useQuery({ queryKey: keys.me, queryFn: api.auth.getMe, retry: false, staleTime: 30_000 })

/** Creates the first administrator and signs them in. */
export const useRunSetup = (cb?: MutationCallbacks<LoginResponse, Parameters<typeof api.setup.run>[0]>) =>
  useApiMutation(
    api.setup.run,
    (qc, res) => {
      qc.setQueryData(keys.me, res.user)
      qc.setQueryData(keys.setup, { required: false })
    },
    cb,
  )

/** Signs in; the answer is the signed-in user. */
export const useLogin = (cb?: MutationCallbacks<LoginResponse, { username: string; password: string }>) =>
  useApiMutation(
    ({ username, password }) => api.auth.login(username, password),
    (qc, res) => qc.setQueryData(keys.me, res.user),
    cb,
  )

/** Signs out and forgets all cached data (also when the session was gone already); the caller navigates. */
export const useLogout = (cb?: MutationCallbacks<void, void>) =>
  useApiMutation(
    () => api.auth.logout().catch(() => {}),
    (qc) => qc.clear(),
    cb,
  )

/** Ends every session of the user, including this one. */
export const useLogoutAll = (cb?: MutationCallbacks<void, void>) =>
  useApiMutation(api.auth.logoutAll, (qc) => qc.clear(), cb)

/** Changing the password ends all sessions (revokeTokens: also the API tokens); the caller opens the sign-in page. */
export const useUpdatePassword = (
  cb?: MutationCallbacks<void, { current: string; next: string; revokeTokens: boolean }>,
) =>
  useApiMutation(
    ({ current, next, revokeTokens }) => api.auth.updatePassword(current, next, revokeTokens),
    (qc) => qc.clear(),
    cb,
  )

/** Calls onEnd when a request finds the session gone (after clearing the cache). */
export function useSessionEnd(onEnd: () => void) {
  const qc = useQueryClient()
  useEffect(() => {
    setUnauthorizedHandler(() => {
      qc.clear()
      onEnd()
    })
    return () => setUnauthorizedHandler(null)
  }, [qc, onEnd])
}

/** The API tokens of the signed-in user. */
export const useTokens = () => useQuery({ queryKey: keys.tokens, queryFn: api.auth.listTokens, staleTime: 60_000 })

// Updated from the answer: the panel reads the list from its lagging cache. A request still running is cancelled
// first, its older answer would overwrite the update.
const updateTokens = (qc: QueryClient, update: (list: TokenView[]) => TokenView[]) => {
  void qc.cancelQueries({ queryKey: keys.tokens })
  qc.setQueryData<TokenView[]>(keys.tokens, (list = []) => update(list))
}

/** Creates an API token and adds it to the list from the answer. */
export const useCreateToken = (cb?: MutationCallbacks<CreatedToken, { name: string; expiresInDays: number }>) =>
  useApiMutation(
    ({ name, expiresInDays }) => api.auth.createToken(name, expiresInDays),
    (qc, { id, name, createdAt, expiresAt }) =>
      updateTokens(qc, (list) => [...list, { id, name, createdAt, expiresAt }]),
    cb,
  )

/** Revokes an API token and removes it from the list. */
export const useDeleteToken = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(api.auth.deleteToken, (qc, _, id) => updateTokens(qc, (list) => list.filter((t) => t.id !== id)), cb)
