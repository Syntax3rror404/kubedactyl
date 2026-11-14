import { useEffect, useRef, type ComponentProps } from "react"

import { TabsList } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

/**
 * A tab bar that scrolls sideways when its tabs do not fit (phones) instead of overflowing or
 * wrapping; a newly active tab is scrolled into view, so scrolling by hand is kept otherwise.
 */
export function ScrollableTabsList({ className, ...props }: ComponentProps<typeof TabsList>) {
  const ref = useRef<HTMLDivElement>(null)
  const shown = useRef<HTMLElement | null>(null)
  useEffect(() => {
    const bar = ref.current
    const active = bar?.querySelector<HTMLElement>("[data-active]") ?? null
    if (!bar || !active || active === shown.current) return
    shown.current = active
    bar.scrollLeft = active.offsetLeft - (bar.clientWidth - active.clientWidth) / 2
  })
  return (
    <TabsList
      ref={ref}
      className={cn("max-w-full justify-start overflow-x-auto [scrollbar-width:none] *:flex-none", className)}
      {...props}
    />
  )
}
