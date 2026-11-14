import { useState } from "react"

/**
 * The value, or the last one that was not null while it is null: a dialog whose target is cleared on close keeps
 * showing it while it animates out (`const user = useRetained(dialog?.kind === "delete" ? dialog.user : null)`).
 */
export function useRetained<T>(value: T | null): T | null {
  const [last, setLast] = useState(value)
  if (value !== null && value !== last) setLast(value)
  return value ?? last
}
