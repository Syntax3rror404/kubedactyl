import { useEffect, useRef } from "react"
import { createApiReference } from "@scalar/api-reference"
import "@scalar/api-reference/style.css"

import { QueryState } from "@/base/components/query-state"
import { useBrandingHead } from "@/base/hooks/use-branding-head"
import { useResolvedTheme } from "@/base/hooks/use-resolved-theme"
import { useApiDocument } from "@/base/lib/queries"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

/** /api-docs: the API reference of the panel (public, unless an administrator turned it off). */
export function ApiDocsPage() {
  const document = useApiDocument()
  useBrandingHead(["API docs"])
  return (
    // Scalar fills the page; the skeleton and the error get a margin.
    <div className={cn("min-h-svh bg-background", document.data === undefined && "p-4 md:p-8")}>
      <QueryState query={document} skeleton={<Skeleton className="h-[80vh] rounded-2xl" />}>
        {(content) => <ApiReference content={content} />}
      </QueryState>
    </div>
  )
}

// The panel's fonts instead of Scalar's, which come from its CDN.
const customCss = ".scalar-app { --scalar-font: var(--font-sans); --scalar-font-code: var(--font-mono); }"

/**
 * Scalar's API reference for an OpenAPI document. Everything that would go online is off (telemetry, CDN fonts,
 * AI agent, MCP, developer tools); it follows the panel's theme.
 */
function ApiReference({ content }: { content: string }) {
  const element = useRef<HTMLDivElement>(null)
  const theme = useResolvedTheme()
  useEffect(() => {
    if (!element.current) return
    const reference = createApiReference(element.current, {
      content,
      forceDarkModeState: theme,
      hideDarkModeToggle: true,
      withDefaultFonts: false,
      telemetry: false,
      showDeveloperTools: "never",
      agent: { disabled: true },
      mcp: { disabled: true },
      customCss,
    })
    return () => reference.destroy()
  }, [content, theme])
  return <div ref={element} />
}
