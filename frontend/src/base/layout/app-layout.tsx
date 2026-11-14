import { Outlet, useMatches } from "react-router"
import { HourglassIcon } from "lucide-react"

import { Callout } from "@/base/components/callout"
import { WaveDots } from "@/base/components/wave-dots"
import { useThrottle } from "@/base/hooks/use-throttle"
import { RequestRates } from "@/base/layout/request-rates"
import { SiteFooter } from "@/base/layout/site-footer"
import { SiteHeader } from "@/base/layout/site-header"
import { sidebarDefaultOpen } from "@/base/lib/open-state"
import { AppSidebar } from "@/base/navigation/app-sidebar"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

/** Route handle of a page that uses the whole width (wide tables); the others stay at a readable width. */
export type WideHandle = { wide?: boolean }

/** Frame of every signed-in page: sidebar, header with breadcrumbs, content and footer. */
export function AppLayout() {
  const wide = useMatches().some((m) => (m.handle as WideHandle | undefined)?.wide)
  return (
    <SidebarProvider defaultOpen={sidebarDefaultOpen()}>
      <AppSidebar />
      <SidebarInset className="relative overflow-hidden">
        {/* Subtle wavy dot grid over the whole width, full down to the middle of the screen, then fading out */}
        <WaveDots className="absolute inset-x-0 top-0 h-svh w-full text-border [mask-image:linear-gradient(to_bottom,black_50%,transparent_100%)]" />
        <SiteHeader />
        <main className="relative flex-1 p-4 md:p-6 lg:p-8">
          <div className={cn("mx-auto w-full", !wide && "max-w-7xl")}>
            <Outlet />
          </div>
        </main>
        <SiteFooter className="relative pb-4">
          <RequestRates />
        </SiteFooter>
        <ThrottleNotice />
      </SidebarInset>
    </SidebarProvider>
  )
}

/** Floating notice while the panel refuses requests for a moment (Retry-After); pages poll on by themselves. */
function ThrottleNotice() {
  const throttle = useThrottle()
  if (!throttle) return null
  return (
    <Callout
      tone="warning"
      icon={<HourglassIcon />}
      title={throttle.busy ? "The panel is busy" : "Too many requests"}
      progress
      className="fixed inset-x-4 bottom-4 z-50 mx-auto max-w-md bg-background shadow-lg"
    >
      <p className="text-muted-foreground">
        {throttle.busy
          ? "The panel is at its API limit. The page continues on its own in a moment."
          : "You sent too many requests. The page continues on its own in a moment."}
      </p>
    </Callout>
  )
}
