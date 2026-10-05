import { MagicCard } from "@/components/ui/magic-card"
import { useResolvedTheme } from "@/hooks/use-resolved-theme"
import { cn } from "@/lib/utils"

/**
 * Card with the Magic UI spotlight border in the brand colors (emerald → sky, like the logo).
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
      gradientFrom="#10b981"
      gradientTo="#0ea5e9"
      gradientSize={220}
      gradientColor={theme === "dark" ? "#1f2937" : "#e2e8f0"}
      gradientOpacity={theme === "dark" ? 0.55 : 0.45}
    >
      {children}
    </MagicCard>
  )
}
