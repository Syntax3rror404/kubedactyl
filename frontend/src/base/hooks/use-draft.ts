import { useMemo, useState } from "react"

// JSON with sorted object keys, so the order in which fields were set does not make a form dirty.
const canonical = (value: unknown) =>
  JSON.stringify(value, (_, v: unknown) =>
    v && typeof v === "object" && !Array.isArray(v)
      ? Object.fromEntries(Object.entries(v).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)))
      : v,
  )

/**
 * State of a form as one object: `draft` holds the values, `set("name", value)` changes one,
 * `dirty` tells whether anything differs from `initial`, `reset()` goes back to it. Pass a memoized `initial`
 * (e.g. `useMemo(() => toDraft(saved), [saved])`), so it is not serialized again on every keystroke.
 */
export function useDraft<T extends object>(initial: T) {
  const [draft, setDraft] = useState(initial)
  const saved = useMemo(() => canonical(initial), [initial])
  const set = <K extends keyof T>(key: K, value: T[K]) => setDraft((d) => ({ ...d, [key]: value }))
  const dirty = canonical(draft) !== saved
  return { draft, set, setDraft, dirty, reset: () => setDraft(initial) }
}
