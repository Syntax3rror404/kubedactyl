import { BRAND_FROM, BRAND_TO, brandIcon } from "@/app/config"
import { useBranding } from "@/base/lib/queries"
import { cn } from "@/lib/utils"

/** The app's logo: the configured image, or the built-in mark (the app's icon on its brand gradient). */
export function BrandMark({ className, iconClassName }: { className?: string; iconClassName?: string }) {
  const logo = useBranding().data?.logo
  if (logo) {
    return (
      <img src={logo} alt="" className={cn("aspect-square size-8 shrink-0 rounded-lg object-contain", className)} />
    )
  }
  const { icon: Icon, ...iconProps } = brandIcon
  return (
    <div
      className={cn(
        "flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg text-white shadow-md",
        className,
      )}
      style={{
        backgroundImage: `linear-gradient(to bottom right, ${BRAND_FROM}, ${BRAND_TO})`,
        boxShadow: `0 4px 6px -1px ${BRAND_FROM}33`,
      }}
    >
      <Icon className={cn("size-4", iconClassName)} {...iconProps} />
    </div>
  )
}
