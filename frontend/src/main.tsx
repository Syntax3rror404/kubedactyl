import { StrictMode } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createRoot } from "react-dom/client"
import { RouterProvider } from "react-router"

import { AppToaster } from "@/app/app-toaster"
import { router } from "@/app/router"
import { ThemeProvider } from "@/components/theme-provider"
import { TooltipProvider } from "@/components/ui/tooltip"
import "@/index.css"
import { ApiError } from "@/lib/api"

// Queries retry once, but not answers that would not change (4xx); 429 means "try again later".
const retry = (failures: number, err: Error) =>
  failures < 1 && !(err instanceof ApiError && err.status >= 400 && err.status < 500 && err.status !== 429)

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry, refetchOnWindowFocus: false } },
})

// After an upgrade an open tab asks for page chunks of its older build, which are gone: load the new build. Once per
// minute at most, so a chunk that is really missing shows the error instead of reloading forever.
window.addEventListener("vite:preloadError", (event) => {
  try {
    const last = Number(sessionStorage.getItem("chunk-reload"))
    if (Date.now() - last < 60_000) return
    sessionStorage.setItem("chunk-reload", String(Date.now()))
  } catch {
    return
  }
  event.preventDefault()
  window.location.reload()
})

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="system" storageKey="app-theme">
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delay={200}>
          <RouterProvider router={router} />
          <AppToaster richColors position="bottom-right" />
        </TooltipProvider>
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
)
