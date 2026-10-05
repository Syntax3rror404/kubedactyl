import { useState } from "react"

/**
 * State of a form as one object: `draft` holds the values, `set("name", value)` changes one,
 * `dirty` tells whether anything differs from `initial`, `reset()` goes back to it.
 */
export function useDraft<T extends object>(initial: T) {
  const [draft, setDraft] = useState(initial)
  const set = <K extends keyof T>(key: K, value: T[K]) => setDraft((d) => ({ ...d, [key]: value }))
  const dirty = JSON.stringify(draft) !== JSON.stringify(initial)
  return { draft, set, setDraft, dirty, reset: () => setDraft(initial) }
}
