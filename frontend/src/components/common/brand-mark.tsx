import { ContainerIcon } from "lucide-react"

import { useBranding } from "@/lib/queries"
import { cn } from "@/lib/utils"

/** The brand gradient (emerald to sky) of the built-in logo, the built-in favicon and GlowCard. */
export const BRAND_FROM = "#00bc7d"
export const BRAND_TO = "#0084d1"

/** The panel's logo: the configured image, or the built-in mark (the container icon on the brand gradient). */
export function BrandMark({ className, iconClassName }: { className?: string; iconClassName?: string }) {
  const logo = useBranding().data?.logo
  if (logo) return <img src={logo} alt="" className={cn("shrink-0 rounded-lg object-contain", className)} />
  return (
    <div
      className={cn(
        "flex shrink-0 items-center justify-center rounded-lg text-white shadow-md shadow-emerald-500/20",
        className,
      )}
      style={{ backgroundImage: `linear-gradient(to bottom right, ${BRAND_FROM}, ${BRAND_TO})` }}
    >
      <ContainerIcon className={iconClassName} />
    </div>
  )
}
