import { accountCards, accountDetail } from "@/app/users"
import { OidcBadge, RoleBadge } from "@/base/components/role-badge"
import { UserAvatar } from "@/base/components/user-avatar"
import { PasswordCard } from "@/base/features/account/components/password-card"
import { SessionsCard } from "@/base/features/account/components/sessions-card"
import { TokensCard } from "@/base/features/account/components/tokens-card"
import { useAuth } from "@/base/hooks/use-auth"
import { PageHeader } from "@/base/layout/page-header"
import { userName } from "@/base/lib/format"

/** /account: password (accounts that have one), sessions, API tokens and the app's cards of the signed-in user. */
export function AccountPage() {
  const { user } = useAuth()
  return (
    <div className="space-y-6">
      <PageHeader
        icon={<UserAvatar user={user} className="size-14 rounded-xl text-lg" />}
        title={userName(user)}
        description={
          <span className="flex flex-wrap items-center gap-2">
            <span>{user.username}</span>
            <RoleBadge role={user.role} />
            {user.oidc && <OidcBadge />}
            {accountDetail(user)}
          </span>
        }
      />
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="space-y-6">
          {user.hasPassword && <PasswordCard />}
          <SessionsCard />
        </div>
        <div className="space-y-6">
          <TokensCard />
          {accountCards.map((Card, i) => (
            <Card key={i} />
          ))}
        </div>
      </div>
    </div>
  )
}
