import { BrandMark } from "@/components/common/brand-mark"
import { Markdown } from "@/components/common/markdown"
import { QueryState } from "@/components/common/query-state"
import { SiteFooter } from "@/components/layout/site-footer"
import { ModeToggle } from "@/components/mode-toggle"
import { Skeleton } from "@/components/ui/skeleton"
import { useBrandingHead } from "@/hooks/use-branding-head"
import { useLicenses } from "@/lib/queries"

/** License texts of the panel and every bundled dependency (public, written by the production build). */
export function LicensesPage() {
  const licenses = useLicenses()
  useBrandingHead(["Licenses"])
  return (
    <div className="min-h-svh bg-background">
      <div className="mx-auto max-w-4xl space-y-6 p-4 md:p-6 lg:p-8">
        <header className="flex items-center gap-3">
          <BrandMark className="size-9 rounded-lg" iconClassName="size-5" />
          <h1 className="flex-1 text-2xl font-semibold tracking-tight">Licenses</h1>
          <ModeToggle />
        </header>
        <QueryState query={licenses} skeleton={<Skeleton className="h-96 rounded-2xl" />}>
          {(text) => <Markdown>{text}</Markdown>}
        </QueryState>
        <SiteFooter />
      </div>
    </div>
  )
}
