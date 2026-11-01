import { useMutation, useQueryClient, type QueryClient, type UseMutationOptions } from "@tanstack/react-query"

/** What a component adds to a mutation: messages, navigation, closing dialogs. */
export type MutationCallbacks<TData, TVars> = Pick<
  UseMutationOptions<TData, Error, TVars>,
  "onSuccess" | "onError" | "onSettled"
>

/**
 * Base of every mutation hook in lib/queries: the hook sends the request and updates the cache
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

/** Marks everything below a group key stale, e.g. keys.allFiles: every listing and opened file of a server. */
export function refreshAll(qc: QueryClient, group: QueryKey) {
  void qc.invalidateQueries({ queryKey: group })
}

/** The cache update of a mutation that only makes the given data stale: `useApiMutation(fn, refreshes(keys.users))`. */
export const refreshes =
  (...queryKeys: QueryKey[]) =>
  (qc: QueryClient) =>
    refresh(qc, ...queryKeys)
