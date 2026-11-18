import { useEffect, useRef } from "react"
import { createApiReference } from "@scalar/api-reference"
import "@scalar/api-reference/style.css"
import { BookOpenIcon } from "lucide-react"

import { EmptyState, QueryState } from "@/base/components/query-state"
import { useBrandingHead } from "@/base/hooks/use-branding-head"
import { useResolvedTheme } from "@/base/hooks/use-resolved-theme"
import { ApiError } from "@/base/lib/api"
import { useApiDocument } from "@/base/lib/queries"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

/** /api-docs: the API reference of the panel (public, unless an administrator turned it off). */
export function ApiDocsPage() {
  const document = useApiDocument()
  useBrandingHead(["API docs"])
  // The panel answers 404 while an administrator has turned the documentation off.
  const off = document.error instanceof ApiError && document.error.status === 404
  return (
    // Scalar fills the page; the skeleton, the error and the note get a margin.
    <div className={cn("min-h-svh bg-background", (off || document.data === undefined) && "p-4 md:p-8")}>
      {off ? (
        <EmptyState
          icon={<BookOpenIcon />}
          title="The API documentation is turned off"
          description="An administrator can turn it on again in the settings."
        />
      ) : (
        <QueryState query={document} skeleton={<Skeleton className="h-[80vh] rounded-2xl" />}>
          {(content) => <ApiReference content={content} />}
        </QueryState>
      )}
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
