import { Fragment } from "react"
import { Link, useMatches } from "react-router"

import { homeLabel, useCrumbLabels } from "@/app/config"
import { ModeToggle } from "@/base/components/mode-toggle"
import { useBrandingHead } from "@/base/hooks/use-branding-head"
import type { Crumb } from "@/base/types"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"

/** Route handle: static crumbs, or a function of the route params. */
export type CrumbHandle = { crumbs: (params: Record<string, string | undefined>) => Crumb[] }

/** Top bar: sidebar toggle, breadcrumbs from the route handles, theme toggle; sets the tab title and favicon. */
export function SiteHeader() {
  const matches = useMatches()
  const crumbs: Crumb[] = []
  for (const m of matches) {
    const handle = m.handle as CrumbHandle | undefined
    if (handle?.crumbs) crumbs.push(...handle.crumbs(m.params))
  }
  const labelled = useCrumbLabels(crumbs)
  // The browser tab follows the breadcrumbs ("Files · Survival · <app name>"), without home on other pages.
  useBrandingHead(
    labelled
      .filter((c) => c.label !== homeLabel || labelled.length === 1)
      .map((c) => c.label)
      .reverse(),
  )
  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4 transition-[width,height] ease-linear">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="mr-2 data-[orientation=vertical]:h-4" />
      <Breadcrumb>
        <BreadcrumbList>
          {labelled.map((c, i) => (
            <Fragment key={i}>
              {i > 0 && <BreadcrumbSeparator />}
              <BreadcrumbItem>
                {c.to && i < labelled.length - 1 ? (
                  <BreadcrumbLink render={<Link to={c.to} />}>{c.label}</BreadcrumbLink>
                ) : (
                  <BreadcrumbPage>{c.label}</BreadcrumbPage>
                )}
              </BreadcrumbItem>
            </Fragment>
          ))}
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto flex items-center gap-2">
        <ModeToggle />
      </div>
    </header>
  )
}
