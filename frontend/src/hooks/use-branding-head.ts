import { useEffect } from "react"

import { useBranding } from "@/lib/queries"

/**
 * Keeps the browser tab in sync with the branding: the title ("Files · Survival · <name>") and the
 * favicon (none unless an administrator uploaded one).
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
    document.querySelector("link[rel=icon]")?.remove()
    if (!favicon) return
    const link = document.createElement("link")
    link.rel = "icon"
    link.href = favicon
    document.head.appendChild(link)
  }, [favicon])
}
