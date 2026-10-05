import type { ReactNode } from "react"

import { BrandMark } from "@/components/common/brand-mark"
import { SiteFooter } from "@/components/layout/site-footer"
import { ModeToggle } from "@/components/mode-toggle"
import { LightRays } from "@/components/ui/light-rays"
import { Particles } from "@/components/ui/particles"
import { useBrandingHead } from "@/hooks/use-branding-head"
import { useResolvedTheme } from "@/hooks/use-resolved-theme"
import { useBranding } from "@/lib/queries"

/** Centered frame of the login and setup pages (logo, Magic UI light rays and particles, theme toggle). */
export function AuthShell({ children, wide, title }: { children: ReactNode; wide?: boolean; title: string }) {
  const theme = useResolvedTheme()
  const branding = useBranding().data
  useBrandingHead([title])
  return (
    <div className="relative flex min-h-svh items-center justify-center overflow-hidden bg-background p-4">
      <LightRays className="opacity-70 dark:opacity-100" color="rgba(16, 185, 129, 0.22)" />
      {/* Remounted on theme changes: the canvas reads the color once. */}
      <Particles
        key={theme}
        className="absolute inset-0"
        quantity={140}
        size={0.6}
        staticity={40}
        ease={70}
        vy={-0.08}
        color={theme === "dark" ? "#6ee7b7" : "#059669"}
      />
      <div className="absolute top-4 right-4">
        <ModeToggle />
      </div>
      <div className={`relative z-10 w-full space-y-6 ${wide ? "max-w-md" : "max-w-sm"}`}>
        <div className="flex flex-col items-center gap-3 text-center">
          <BrandMark className="size-12 rounded-xl" iconClassName="size-6" />
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{branding?.name}</h1>
            <p className="text-sm text-muted-foreground">{branding?.tagline}</p>
          </div>
        </div>
        {children}
      </div>
      <SiteFooter className="absolute inset-x-0 bottom-4 z-10" />
    </div>
  )
}
