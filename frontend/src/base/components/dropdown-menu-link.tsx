import { Menu as MenuPrimitive } from "@base-ui/react/menu"

import { cn } from "@/lib/utils"

/**
 * Menu entry that navigates (Base UI `LinkItem`, an `<a>`), styled like `DropdownMenuItem`; shadcn ships no wrapper
 * for it. Closes the menu on click: the sidebar menu stays mounted when the page changes, a download keeps the page.
 */
export function DropdownMenuLink({ className, closeOnClick = true, ...props }: MenuPrimitive.LinkItem.Props) {
  return (
    <MenuPrimitive.LinkItem
      data-slot="dropdown-menu-item"
      closeOnClick={closeOnClick}
      className={cn(
        // Default variant of DropdownMenuItem (components/ui/dropdown-menu.tsx).
        "relative flex cursor-default items-center gap-1.5 rounded-md px-1.5 py-1 text-sm outline-hidden select-none",
        "focus:bg-accent focus:text-accent-foreground focus:**:text-accent-foreground",
        "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className,
      )}
      {...props}
    />
  )
}
