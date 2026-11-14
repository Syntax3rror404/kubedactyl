import { BrandMark } from "@/base/components/brand-mark"
import { Markdown } from "@/base/components/markdown"
import { ModeToggle } from "@/base/components/mode-toggle"
import { QueryState } from "@/base/components/query-state"
import { useBrandingHead } from "@/base/hooks/use-branding-head"
import { SiteFooter } from "@/base/layout/site-footer"
import { useLicenses } from "@/base/lib/queries"
import { Skeleton } from "@/components/ui/skeleton"

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
