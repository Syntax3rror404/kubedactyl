import { BRAND_FROM, BRAND_TO } from "@/app/config"
import { useResolvedTheme } from "@/base/hooks/use-resolved-theme"
import { MagicCard } from "@/components/ui/magic-card"
import { cn } from "@/lib/utils"

/**
 * Card with the Magic UI spotlight border in the app's brand colors, like the logo.
 * The library card paints --color-background; here it uses the card color and fills the grid cell.
 */
export function GlowCard({ children, className }: { children: React.ReactNode; className?: string }) {
  const theme = useResolvedTheme()
  return (
    <MagicCard
      className={cn(
        "h-full rounded-2xl shadow-sm [--color-background:var(--color-card)] [&>div:last-child]:h-full",
        className,
      )}
      gradientFrom={BRAND_FROM}
      gradientTo={BRAND_TO}
      gradientSize={220}
      gradientColor={theme === "dark" ? "#1f2937" : "#e2e8f0"}
      gradientOpacity={theme === "dark" ? 0.55 : 0.45}
    >
      {children}
    </MagicCard>
  )
}
