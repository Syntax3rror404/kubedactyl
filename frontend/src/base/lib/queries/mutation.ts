import { useMutation, useQueryClient, type QueryClient, type UseMutationOptions } from "@tanstack/react-query"
import { toast } from "sonner"

/** What a component adds to a mutation: messages, navigation, closing dialogs. */
export type MutationCallbacks<TData, TVars> = Pick<
  UseMutationOptions<TData, Error, TVars>,
  "onSuccess" | "onError" | "onSettled"
>

// Toast wording used everywhere: success "<Thing> <past participle>" ("Backup deleted"),
// failure "Could not <action>" with the message of the server as description.

/** onError of a mutation: `useDeleteUser(name, { onError: failed("delete the user") })`. */
export const failed = (action: string) => (err: Error) =>
  toast.error(`Could not ${action}`, { description: err.message })

/**
 * Base of every mutation hook (lib/queries of the base and the app): the hook sends the request and updates the cache
 * (updateCache), the component passes what the user sees as callbacks.
 */
export function useApiMutation<TData, TVars = void>(
  mutationFn: (vars: TVars) => Promise<TData>,
  updateCache: (qc: QueryClient, data: TData, vars: TVars) => void,
  callbacks: MutationCallbacks<TData, TVars> = {},
) {
  const qc = useQueryClient()
  return useMutation<TData, Error, TVars>({
    ...callbacks,
    mutationFn,
    onSuccess: (...args) => {
      updateCache(qc, args[0], args[1])
      return callbacks.onSuccess?.(...args)
    },
  })
}

type QueryKey = readonly unknown[]

/** Marks the cached data of exactly these keys stale (never the keys below them); mounted queries load it again. */
export function refresh(qc: QueryClient, ...queryKeys: QueryKey[]) {
  for (const queryKey of queryKeys) void qc.invalidateQueries({ queryKey, exact: true })
}

/** Marks everything below group keys stale, e.g. every listing and opened file of one object. */
export function refreshAll(qc: QueryClient, ...groups: QueryKey[]) {
  for (const queryKey of groups) void qc.invalidateQueries({ queryKey })
}

/** The cache update of a mutation that only makes the given data stale: `useApiMutation(fn, refreshes(keys.users))`. */
export const refreshes =
  (...queryKeys: QueryKey[]) =>
  (qc: QueryClient) =>
    refresh(qc, ...queryKeys)
