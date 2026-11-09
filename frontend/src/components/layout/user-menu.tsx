import { ChevronsUpDownIcon, LogOutIcon, UserCogIcon } from "lucide-react"
import { Link, useNavigate } from "react-router"

import { DropdownMenuLink } from "@/components/common/dropdown-menu-link"
import { UserAvatar } from "@/components/common/user-avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar"
import { useAuth } from "@/hooks/use-auth"
import { roleLabel, userName } from "@/lib/format"
import { useLogout } from "@/lib/queries"

/** Signed-in user in the sidebar footer: account settings and sign out. */
export function UserMenu() {
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
                    <span className="truncate text-xs text-muted-foreground">{user.email || user.namespace}</span>
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
