import { OidcBadge, RoleBadge } from "@/components/common/role-badge"
import { UserAvatar } from "@/components/common/user-avatar"
import { PageHeader } from "@/components/layout/page-header"
import { PasswordCard } from "@/features/account/components/password-card"
import { SessionsCard } from "@/features/account/components/sessions-card"
import { TokensCard } from "@/features/account/components/tokens-card"
import { useAuth } from "@/hooks/use-auth"
import { userName } from "@/lib/format"

/** /account: the signed-in user: password (accounts that have one), sessions and API tokens. */
export function AccountPage() {
  const { user } = useAuth()
  return (
    <div className="space-y-8">
      <PageHeader
        icon={<UserAvatar user={user} className="size-14 rounded-xl text-lg" />}
        title={userName(user)}
        description={
          <span className="flex flex-wrap items-center gap-2">
            <span className="font-mono">{user.username}</span>
            <RoleBadge role={user.role} />
            {user.oidc && <OidcBadge />}
            <span className="font-mono text-xs">{user.namespace}</span>
          </span>
        }
      />
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="space-y-6">
          {user.hasPassword && <PasswordCard />}
          <SessionsCard />
        </div>
        <TokensCard />
      </div>
    </div>
  )
}
