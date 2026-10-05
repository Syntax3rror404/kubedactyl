import { useEffect } from "react"

/** Asks the browser to confirm closing or reloading the tab while `when` is true (unsaved changes). */
export function useBeforeUnload(when: boolean) {
  useEffect(() => {
    if (!when) return
    const warn = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener("beforeunload", warn)
    return () => window.removeEventListener("beforeunload", warn)
  }, [when])
}
