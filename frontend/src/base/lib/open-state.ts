// What the user opened and closed in the navigation, kept in this browser so a reload shows it the same way.
// Storage can be missing (private windows, blocked site data): the navigation then starts as without it.

const PREFIX = "navigation-open:"

/** The open state the user chose for a part of the navigation; undefined when never chosen. */
export function readOpen(key: string): boolean | undefined {
  try {
    const value = localStorage.getItem(PREFIX + key)
    return value === null ? undefined : value === "1"
  } catch {
    return undefined
  }
}

/** Keeps the open state the user chose. */
export function saveOpen(key: string, open: boolean) {
  try {
    localStorage.setItem(PREFIX + key, open ? "1" : "0")
  } catch {
    // Not kept: the next reload starts with the default again.
  }
}

/** The sidebar as the user left it: the shadcn sidebar writes this cookie but reads it only in Next.js. */
export function sidebarDefaultOpen(): boolean {
  return !document.cookie.split("; ").includes("sidebar_state=false")
}
