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

/** Marks cached data as stale; mounted queries load it again. */
export function refresh(qc: QueryClient, ...queryKeys: readonly (readonly unknown[])[]) {
  for (const queryKey of queryKeys) void qc.invalidateQueries({ queryKey })
}
