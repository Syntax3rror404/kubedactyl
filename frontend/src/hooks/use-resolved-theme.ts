import { useEffect, useState } from "react"

/** Returns the effective theme by observing the class on <html> (set by ThemeProvider). */
export function useResolvedTheme(): "light" | "dark" {
  const get = () => (document.documentElement.classList.contains("dark") ? "dark" : "light")
  const [theme, setTheme] = useState<"light" | "dark">(get)
  useEffect(() => {
    const obs = new MutationObserver(() => setTheme(get()))
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] })
    return () => obs.disconnect()
  }, [])
  return theme
}
