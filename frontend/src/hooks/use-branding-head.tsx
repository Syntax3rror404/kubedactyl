import { useEffect } from "react"
import { flushSync } from "react-dom"
import { createRoot } from "react-dom/client"
import { ContainerIcon } from "lucide-react"

import { useBranding } from "@/lib/queries"

/**
 * Keeps the browser tab in sync with the branding: the title ("Files · Survival · <name>") and the
 * favicon (the uploaded one, or the built-in container icon).
 */
export function useBrandingHead(parts: string[]) {
  const { data } = useBranding()
  const name = data?.name ?? "Kubedactyl"
  const favicon = data?.favicon
  const title = [...parts, name].join(" · ")
  useEffect(() => {
    document.title = title
  }, [title])
  useEffect(() => {
    if (!data) return // the default would flash before an uploaded favicon
    document.querySelector("link[rel=icon]")?.remove()
    const link = document.createElement("link")
    link.rel = "icon"
    link.href = favicon || builtInFavicon
    document.head.appendChild(link)
  }, [data, favicon])
}

/**
 * The built-in favicon: the lucide container icon on the brand gradient (like the built-in logo), as an SVG data
 * URL. Rendered once when the module loads: flushSync does not render from inside an effect.
 */
const builtInFavicon = (() => {
  const div = document.createElement("div")
  const root = createRoot(div)
  flushSync(() =>
    root.render(
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
        <defs>
          <linearGradient id="brand" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor="#00bc7d" />
            <stop offset="1" stopColor="#0084d1" />
          </linearGradient>
        </defs>
        <rect width="32" height="32" rx="8" fill="url(#brand)" />
        <ContainerIcon x={6} y={6} size={20} color="#fff" strokeWidth={2.25} />
      </svg>,
    ),
  )
  const url = "data:image/svg+xml," + encodeURIComponent(div.innerHTML)
  root.unmount()
  return url
})()
