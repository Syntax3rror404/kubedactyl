import { useState } from "react"
import { Link } from "react-router"
import { ChevronRightIcon } from "lucide-react"

import { readOpen, saveOpen } from "@/base/lib/open-state"
import { NavBadgeView, NavMark } from "@/base/navigation/nav-badge"
import { type NavGroup, type NavItem } from "@/base/types"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

// Built like the shadcn block sidebar-07 (nav-main.tsx). The chevron turns by the group on its trigger, not on the
// Collapsible as in the block: nested branches would otherwise turn with every open parent.

/** The entry or one below it is the current page. */
const holdsActive = (item: NavItem): boolean => !!item.active || !!item.children?.some(holdsActive)

/**
 * Open state of a branch: open while it holds the current page, otherwise as the user left it (also after a reload)
 * or as the app starts it. A branch that comes to hold the current page opens without changing what is kept.
 */
function useBranchOpen(item: NavItem): [boolean, (open: boolean) => void] {
  const holds = holdsActive(item)
  const [open, setOpen] = useState(() => holds || (readOpen(item.label) ?? !!item.open))
  const [held, setHeld] = useState(holds)
  if (holds !== held) {
    setHeld(holds)
    if (holds) setOpen(true)
  }
  const choose = (next: boolean) => {
    setOpen(next)
    saveOpen(item.label, next)
  }
  return [open, choose]
}

/**
 * The groups of the sidebar. A branch (an entry with children) opens when it comes to hold the current page and
 * otherwise keeps what the user chose, so moving to another page never closes a part someone is working in.
 */
export function NavTree({ groups, className }: { groups: NavGroup[]; className?: string }) {
  return groups.map((g, i) => (
    <SidebarGroup key={g.label ?? i} className={cn(!g.label && "pb-0", className)}>
      {g.label && <SidebarGroupLabel>{g.label}</SidebarGroupLabel>}
      <SidebarMenu>
        {g.items.map((item) =>
          item.children ? <TopBranch key={item.label} item={item} /> : <TopLink key={item.label} item={item} />,
        )}
      </SidebarMenu>
    </SidebarGroup>
  ))
}

/** Closes the sidebar sheet on phones after a page was picked. */
function useCloseOnPick() {
  const { isMobile, setOpenMobile } = useSidebar()
  return () => isMobile && setOpenMobile(false)
}

function target(item: NavItem, onPick: () => void) {
  if (item.href) return <a href={item.href} target="_blank" rel="noreferrer" />
  return <Link to={item.to ?? "#"} onClick={onPick} />
}

/**
 * Icon (on a colored tile with an accent) and label; inline badges follow for entries without a badge slot. The tile
 * needs size-3.5! like the shadcn Badge: the sidebar buttons size every svg inside them.
 */
function Label({ item, inlineBadge }: { item: NavItem; inlineBadge?: boolean }) {
  return (
    <>
      {item.icon &&
        (item.accent ? (
          <span
            className={cn("flex size-6 shrink-0 items-center justify-center rounded-md [&_svg]:size-3.5!", item.accent)}
          >
            <item.icon />
          </span>
        ) : (
          <item.icon />
        ))}
      <span className={item.mono ? "truncate font-mono" : "truncate"}>{item.label}</span>
      {inlineBadge && item.badge && <NavBadgeView badge={item.badge} className="ml-auto" />}
    </>
  )
}

function TopLink({ item }: { item: NavItem }) {
  const onPick = useCloseOnPick()
  const mark = item.badge?.kind === "text" ? undefined : item.badge
  return (
    <SidebarMenuItem>
      <SidebarMenuButton tooltip={item.label} isActive={item.active} render={target(item, onPick)}>
        <Label item={item} />
      </SidebarMenuButton>
      {item.badge && (
        <SidebarMenuBadge>
          <NavBadgeView badge={item.badge} />
        </SidebarMenuBadge>
      )}
      {mark && (
        // Collapsed to icons the badge is hidden: a mark on the icon remains.
        <NavMark
          tone={mark.tone}
          icon={mark.kind === "status" ? mark.icon : undefined}
          className="pointer-events-none absolute top-1 left-5 hidden group-data-[collapsible=icon]:block"
        />
      )}
    </SidebarMenuItem>
  )
}

function TopBranch({ item }: { item: NavItem }) {
  const { state, isMobile, setOpen: setSidebarOpen } = useSidebar()
  const [open, setOpen] = useBranchOpen(item)
  return (
    <Collapsible open={open} onOpenChange={setOpen} render={<SidebarMenuItem />}>
      <CollapsibleTrigger
        render={<SidebarMenuButton tooltip={item.label} className="group/branch" />}
        // Collapsed to icons the children are hidden: a click widens the sidebar first.
        onClick={() => state === "collapsed" && !isMobile && setSidebarOpen(true)}
      >
        <Label item={item} inlineBadge />
        <BranchChevron item={item} />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <SubList item={item} />
      </CollapsibleContent>
    </Collapsible>
  )
}

/** Turns while the branch is open; right-aligned also without a badge. */
function BranchChevron({ item }: { item: NavItem }) {
  return (
    <ChevronRightIcon
      className={cn(
        "shrink-0 text-muted-foreground transition-transform duration-200 group-data-panel-open/branch:rotate-90",
        !item.badge && "ml-auto",
      )}
    />
  )
}

function SubList({ item }: { item: NavItem }) {
  const items = item.children ?? []
  return (
    <SidebarMenuSub>
      {items.length === 0 && item.empty && <li className="px-2 py-1 text-xs text-muted-foreground">{item.empty}</li>}
      {items.map((child) => (
        <SidebarMenuSubItem key={child.label}>
          {child.children ? <SubBranch item={child} /> : <SubLink item={child} />}
        </SidebarMenuSubItem>
      ))}
    </SidebarMenuSub>
  )
}

function SubLink({ item }: { item: NavItem }) {
  const onPick = useCloseOnPick()
  return (
    <SidebarMenuSubButton isActive={item.active} render={target(item, onPick)}>
      <Label item={item} inlineBadge />
    </SidebarMenuSubButton>
  )
}

function SubBranch({ item }: { item: NavItem }) {
  const [open, setOpen] = useBranchOpen(item)
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        render={<SidebarMenuSubButton render={<button type="button" />} className="group/branch w-full" />}
      >
        <Label item={item} inlineBadge />
        <BranchChevron item={item} />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <SubList item={item} />
      </CollapsibleContent>
    </Collapsible>
  )
}
