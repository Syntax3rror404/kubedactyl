// Data layer of the frontend. Every request goes through a hook from this folder (the base API) or the app's
// lib/queries (its own areas):
//
//   - queries:   useX(...)           → { data, isLoading, error, … }   (reads, polling)
//   - mutations: useDoX(..., cb)     → { mutate, isPending, … }        (writes)
//
// A mutation hook sends the request and updates the cache (setQueryData / invalidate); the
// component passes what the user sees (toasts, navigation, closing a dialog) as callbacks.
// Components never call the API or the query client themselves (oxlint: no-restricted-imports).

export { failed, type MutationCallbacks } from "@/base/lib/queries/mutation"
export * from "@/base/lib/queries/auth"
export * from "@/base/lib/queries/settings"
export * from "@/base/lib/queries/users"
