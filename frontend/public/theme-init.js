// Applies the theme before the first render to avoid a flash of the wrong theme.
// A separate file (not inline) so the Content-Security-Policy can forbid inline scripts.
try {
  const t = localStorage.getItem("app-theme") || "system"
  const dark = t === "dark" || (t === "system" && matchMedia("(prefers-color-scheme: dark)").matches)
  document.documentElement.classList.add(dark ? "dark" : "light")
} catch {
  // storage not available
}
