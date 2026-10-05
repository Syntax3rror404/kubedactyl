import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { initials } from "@/lib/format"
import type { UserView } from "@/lib/types"
import { cn } from "@/lib/utils"

/** Initials avatar of a user. */
export function UserAvatar({
  user,
  className,
}: {
  user: Pick<UserView, "username" | "displayName" | "role">
  className?: string
}) {
  return (
    <Avatar className={cn("size-8 rounded-lg", className)}>
      <AvatarFallback
        className={cn(
          "rounded-lg text-xs font-semibold text-white",
          user.role === "admin"
            ? "bg-gradient-to-br from-fuchsia-500 to-indigo-600"
            : "bg-gradient-to-br from-emerald-500 to-sky-600",
        )}
      >
        {initials(user)}
      </AvatarFallback>
    </Avatar>
  )
}
