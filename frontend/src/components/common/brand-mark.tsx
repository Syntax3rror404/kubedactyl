import { BoxesIcon } from "lucide-react"

import { useBranding } from "@/lib/queries"
import { cn } from "@/lib/utils"

/** The panel's logo: the configured image, or the built-in mark. */
export function BrandMark({ className, iconClassName }: { className?: string; iconClassName?: string }) {
  const logo = useBranding().data?.logo
  if (logo) return <img src={logo} alt="" className={cn("shrink-0 rounded-lg object-contain", className)} />
  return (
    <div
      className={cn(
        "flex shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-emerald-500 to-sky-600 text-white shadow-md shadow-emerald-500/20",
        className,
      )}
    >
      <BoxesIcon className={iconClassName} />
    </div>
  )
}
