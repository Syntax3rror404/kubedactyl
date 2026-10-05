import { useEffect, useRef, type ComponentProps } from "react"

import { TabsList } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

/**
 * A tab bar that scrolls sideways when its tabs do not fit (phones) instead of overflowing or
 * wrapping; the active tab is scrolled into view.
 */
export function ScrollableTabsList({ className, ...props }: ComponentProps<typeof TabsList>) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const active = ref.current?.querySelector<HTMLElement>("[data-state=active],[data-active]")
    if (active && ref.current) {
      const bar = ref.current
      bar.scrollLeft = active.offsetLeft - (bar.clientWidth - active.clientWidth) / 2
    }
  })
  return (
    <TabsList
      ref={ref}
      className={cn("max-w-full justify-start overflow-x-auto [scrollbar-width:none] *:flex-none", className)}
      {...props}
    />
  )
}
