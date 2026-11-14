import { useTheme } from "@/components/theme-provider"
import { Toaster } from "@/components/ui/sonner"

/**
 * The registry Toaster reads the theme from next-themes, which this Vite app does not
 * use. This wrapper passes the theme of our ThemeProvider instead, so ui/sonner.tsx can
 * stay exactly as the registry ships it.
 */
export function AppToaster(props: React.ComponentProps<typeof Toaster>) {
  const { theme } = useTheme()
  return <Toaster theme={theme} {...props} />
}
