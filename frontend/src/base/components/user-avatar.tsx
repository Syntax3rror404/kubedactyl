import { BRAND_FROM, BRAND_TO } from "@/app/config"
import { initials } from "@/base/lib/format"
import type { UserView } from "@/base/lib/types"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
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
          user.role === "admin" && "bg-linear-to-br from-fuchsia-500 to-indigo-600",
        )}
        // Users in the brand colors of the app, administrators in their own.
        style={
          user.role === "admin"
            ? undefined
            : { backgroundImage: `linear-gradient(to bottom right, ${BRAND_FROM}, ${BRAND_TO})` }
        }
      >
        {initials(user)}
      </AvatarFallback>
    </Avatar>
  )
}
