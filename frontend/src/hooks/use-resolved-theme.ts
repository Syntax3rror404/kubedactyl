import { useSyncExternalStore } from "react"

type Theme = "light" | "dark"

const getTheme = (): Theme => (document.documentElement.classList.contains("dark") ? "dark" : "light")

// One observer of the class on <html>, shared by every component that follows the theme (one per card would add up).
const listeners = new Set<() => void>()
let observer: MutationObserver | null = null

function subscribe(listener: () => void) {
  listeners.add(listener)
  if (!observer) {
    observer = new MutationObserver(() => listeners.forEach((l) => l()))
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] })
  }
  return () => {
    listeners.delete(listener)
    if (listeners.size === 0) {
      observer?.disconnect()
      observer = null
    }
  }
}

/** The effective theme: the class ThemeProvider sets on <html>. */
export function useResolvedTheme(): Theme {
  return useSyncExternalStore(subscribe, getTheme)
}
