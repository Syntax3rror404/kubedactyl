import { Outlet } from "react-router"

import { WaveDots } from "@/components/common/wave-dots"
import { AppSidebar } from "@/components/layout/app-sidebar"
import { RequestRates } from "@/components/layout/request-rates"
import { SiteFooter } from "@/components/layout/site-footer"
import { SiteHeader } from "@/components/layout/site-header"
import { ThrottleNotice } from "@/components/layout/throttle-notice"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"

/** Frame of every signed-in page: sidebar, header with breadcrumbs, content and footer. */
export function AppLayout() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="relative overflow-hidden">
        {/* Subtle wavy dot grid over the whole width, full down to the middle of the screen, then fading out */}
        <WaveDots className="absolute inset-x-0 top-0 h-svh w-full text-border [mask-image:linear-gradient(to_bottom,black_50%,transparent_100%)]" />
        <SiteHeader />
        <main className="relative flex-1 p-4 md:p-6 lg:p-8">
          <div className="mx-auto w-full max-w-7xl">
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
