import { useSearchParams } from "react-router"

import { OidcBadge, RoleBadge } from "@/components/common/role-badge"
import { UserAvatar } from "@/components/common/user-avatar"
import { PageHeader } from "@/components/layout/page-header"
import { PasswordCard } from "@/features/account/components/password-card"
import { SessionsCard } from "@/features/account/components/sessions-card"
import { SubusersCard } from "@/features/account/components/subusers-card"
import { TokensCard } from "@/features/account/components/tokens-card"
import { useAuth } from "@/hooks/use-auth"
import { userName } from "@/lib/format"

/**
 * /account: password (accounts that have one), sessions, API tokens and the subusers of the signed-in user.
 * ?invite=<server> (the Invite button of a server) opens the invite of a subuser with that server.
 */
export function AccountPage() {
  const { user } = useAuth()
  const [params, setParams] = useSearchParams()
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
            {user.owner ? (
              <span className="text-xs">Subuser of {user.owner}</span>
            ) : (
              <span className="font-mono text-xs">{user.namespace}</span>
            )}
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
          {!user.owner && (
            <SubusersCard inviteServer={params.get("invite") ?? undefined} onInvited={() => setParams({})} />
          )}
        </div>
      </div>
    </div>
  )
}
