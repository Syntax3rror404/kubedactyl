// Data layer of the frontend. Every request goes through a hook from this folder:
//
//   - queries:   useX(...)           → { data, isLoading, error, … }   (reads, polling)
//   - mutations: useDoX(..., cb)     → { mutate, isPending, … }        (writes)
//
// A mutation hook sends the request and updates the cache (setQueryData / invalidate); the
// component passes what the user sees (toasts, navigation, closing a dialog) as callbacks.
// Components never call `api` or the query client themselves (oxlint: no-restricted-imports).

export type { MutationCallbacks } from "@/lib/queries/mutation"
export * from "@/lib/queries/auth"
export * from "@/lib/queries/panel"
export * from "@/lib/queries/cluster"
export * from "@/lib/queries/users"
export * from "@/lib/queries/eggs"
export * from "@/lib/queries/servers"
export * from "@/lib/queries/schedules"
export * from "@/lib/queries/backups"
export * from "@/lib/queries/files"
