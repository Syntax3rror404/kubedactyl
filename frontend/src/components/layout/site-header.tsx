import { Fragment } from "react"
import { Link, useMatches } from "react-router"

import { ModeToggle } from "@/components/mode-toggle"
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
import { useBrandingHead } from "@/hooks/use-branding-head"
import { serverName } from "@/lib/format"
import { useEgg, useServer } from "@/lib/queries"

interface Crumb {
  label: string
  to?: string
}

/** Route handle: static crumbs, or a function of the route params. */
export type CrumbHandle = { crumbs: (params: Record<string, string | undefined>) => Crumb[] }

/** The crumbs with display names: labels like "server:<name>" or "egg:<name>" resolve to the object's name. */
function useCrumbLabels(crumbs: Crumb[]): Crumb[] {
  const serverRef = crumbs.find((c) => c.label.startsWith("server:"))?.label.slice(7) ?? ""
  const eggRef = crumbs.find((c) => c.label.startsWith("egg:"))?.label.slice(4) ?? ""
  const server = useServer(serverRef)
  const egg = useEgg(eggRef)
  return crumbs.map((c) => {
    if (c.label.startsWith("server:")) return { ...c, label: server.data ? serverName(server.data) : serverRef }
    if (c.label.startsWith("egg:")) return { ...c, label: egg.data?.spec.displayName ?? eggRef }
    return c
  })
}

/** Top bar: sidebar toggle, breadcrumbs from the route handles, theme toggle; sets the tab title and favicon. */
export function SiteHeader() {
  const matches = useMatches()
  const crumbs: Crumb[] = []
  for (const m of matches) {
    const handle = m.handle as CrumbHandle | undefined
    if (handle?.crumbs) crumbs.push(...handle.crumbs(m.params))
  }
  const labelled = useCrumbLabels(crumbs)
  // The browser tab follows the breadcrumbs ("Files · Survival · <panel name>").
  useBrandingHead(
    labelled
      .filter((c) => c.label !== "Dashboard" || labelled.length === 1)
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
                  <BreadcrumbLink asChild>
                    <Link to={c.to}>{c.label}</Link>
                  </BreadcrumbLink>
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
