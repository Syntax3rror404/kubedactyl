import { useEffect, useState } from "react"
import { Link, useNavigate } from "react-router"
import { ChevronsUpDownIcon, LogOutIcon, SearchIcon, UserCogIcon } from "lucide-react"

import { useNavigation } from "@/app/use-navigation"
import { DropdownMenuLink } from "@/base/components/dropdown-menu-link"
import { UserAvatar } from "@/base/components/user-avatar"
import { useAuth } from "@/base/hooks/use-auth"
import { roleLabel, userName } from "@/base/lib/format"
import { useLogout } from "@/base/lib/queries"
import { NavHeader } from "@/base/navigation/nav-header"
import { NavTree } from "@/base/navigation/nav-tree"
import type { NavConfig, NavGroup, NavItem, NavSearchGroup } from "@/base/types"
import {
  Combobox,
  ComboboxCollection,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
} from "@/components/ui/combobox"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Kbd } from "@/components/ui/kbd"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar"

// The search key as the keyboard shows it: Command on Apple devices, Ctrl elsewhere (both open the search).
const SEARCH_KEY = /Mac|iPhone|iPad/.test(navigator.userAgent) ? "⌘K" : "Ctrl K"

/**
 * The sidebar of every app: header, search, the app's groups, the footer group and the user menu. What it shows
 * comes from useNavigation (app/use-navigation.ts); this file stays the same in all apps.
 */
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const config = useNavigation()
  const groups = config.footer ? [...config.groups, config.footer] : config.groups
  const [search, setSearch] = useState(false)
  return (
    <Sidebar collapsible="icon" variant="inset" {...props}>
      <SidebarHeader>
        <NavHeader header={config.header} />
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip="Search"
              variant="outline"
              className="text-muted-foreground"
              onClick={() => setSearch(true)}
            >
              <SearchIcon />
              <span>Search</span>
              <Kbd className="ml-auto">{SEARCH_KEY}</Kbd>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavTree groups={config.groups} />
        {config.footer && <NavTree groups={[config.footer]} className="mt-auto" />}
      </SidebarContent>
      <SidebarFooter>
        <UserMenu config={config} />
      </SidebarFooter>
      <SidebarRail />
      <CommandSearch groups={groups} search={config.search} open={search} onOpenChange={setSearch} />
    </Sidebar>
  )
}

/** Signed-in user in the sidebar footer: account settings and sign out. */
function UserMenu({ config }: { config: Pick<NavConfig, "userDetail"> }) {
  const { user } = useAuth()
  const navigate = useNavigate()
  const logout = useLogout({ onSuccess: () => navigate("/login", { replace: true }) })
  const { isMobile } = useSidebar()
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size="lg"
                className="data-popup-open:bg-sidebar-accent data-popup-open:text-sidebar-accent-foreground"
              />
            }
          >
            <UserAvatar user={user} />
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">{userName(user)}</span>
              <span className="truncate text-xs text-muted-foreground">
                {user.role === "admin" ? roleLabel(user.role) : user.username}
              </span>
            </div>
            <ChevronsUpDownIcon className="ml-auto size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className="p-0 font-normal">
                <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                  <UserAvatar user={user} />
                  <div className="grid flex-1 leading-tight">
                    <span className="truncate font-medium">{userName(user)}</span>
                    <span className="truncate text-xs text-muted-foreground">
                      {user.email || config.userDetail || roleLabel(user.role)}
                    </span>
                  </div>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuLink render={<Link to="/account" />}>
              <UserCogIcon />
              Account
            </DropdownMenuLink>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled={logout.isPending} onClick={() => logout.mutate()}>
              <LogOutIcon />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

type Entry = NavSearchGroup["items"][number]

/** Every page of the sidebar; the hint names where it sits ("Networking"). */
function pages(items: NavItem[], parent = ""): Entry[] {
  return items.flatMap((item) => [
    ...(item.to ? [{ label: item.label, to: item.to, hint: parent }] : []),
    ...pages(item.children ?? [], item.label),
  ])
}

/** The groups of the search: pages first, then the app's own entries without the pages already listed. */
function searchGroups(groups: NavGroup[], search: NavSearchGroup[]): NavSearchGroup[] {
  const own = pages(groups.flatMap((g) => g.items))
  const seen = new Set(own.map((e) => e.to))
  const rest = search.map((g) => ({ ...g, items: g.items.filter((e) => !seen.has(e.to)) }))
  return [{ label: "Pages", items: own }, ...rest].filter((g) => g.items.length > 0)
}

/** ⌘K or Ctrl+K anywhere: pages, objects and types of the app, found by name and hint. */
function CommandSearch({
  groups,
  search,
  open,
  onOpenChange,
}: {
  groups: NavGroup[]
  search: NavSearchGroup[]
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const navigate = useNavigate()
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        onOpenChange(true)
      }
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [onOpenChange])
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="top-1/3 translate-y-0 gap-0 overflow-hidden p-0" showCloseButton={false}>
        <DialogHeader className="sr-only">
          <DialogTitle>Search</DialogTitle>
          <DialogDescription>Pages, objects and types</DialogDescription>
        </DialogHeader>
        <Combobox
          inline
          open
          autoHighlight
          items={open ? searchGroups(groups, search) : []}
          value={null}
          onValueChange={(e: Entry | null) => {
            if (!e) return
            onOpenChange(false)
            navigate(e.to)
          }}
          itemToStringLabel={(e: Entry) => e.label}
          filter={(e: Entry, query) => `${e.label} ${e.hint ?? ""}`.toLowerCase().includes(query.toLowerCase())}
        >
          <ComboboxInput showTrigger={false} placeholder="Search pages, objects and types…" className="m-1" />
          <ComboboxEmpty className="flex empty:py-0">Nothing found.</ComboboxEmpty>
          <ComboboxList className="max-h-80">
            {(g: NavSearchGroup) => (
              <ComboboxGroup key={g.label} items={g.items}>
                <ComboboxLabel>{g.label}</ComboboxLabel>
                <ComboboxCollection>
                  {(e: Entry) => (
                    <ComboboxItem key={e.to} value={e}>
                      <span className="truncate">{e.label}</span>
                      {e.hint && <span className="ml-auto truncate text-xs text-muted-foreground">{e.hint}</span>}
                    </ComboboxItem>
                  )}
                </ComboboxCollection>
              </ComboboxGroup>
            )}
          </ComboboxList>
        </Combobox>
      </DialogContent>
    </Dialog>
  )
}
