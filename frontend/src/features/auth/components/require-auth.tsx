import { useCallback, useMemo } from "react"
import { Navigate, Outlet, useLocation, useNavigate } from "react-router"

import { ErrorState } from "@/components/common/query-state"
import { Spinner } from "@/components/ui/spinner"
import { AuthContext, useAuth } from "@/hooks/use-auth"
import { ApiError } from "@/lib/api"
import { useMe, useSessionEnd, useSetupStatus } from "@/lib/queries"

/**
 * Loads the current user; without a session (401) the visitor is sent to /login (or /setup on the first start), other
 * failures can be retried. A user whose password an administrator set chooses an own one first (/change-password).
 */
export function RequireAuth() {
  const me = useMe()
  const signedOut = me.error instanceof ApiError && me.error.status === 401
  const setup = useSetupStatus(signedOut)
  const navigate = useNavigate()
  const location = useLocation()
  const next = location.pathname + location.search
  // The same object while the user is unchanged, so useAuth() readers update only when the user does.
  const auth = useMemo(() => me.data && { user: me.data, isAdmin: me.data.role === "admin" }, [me.data])

  useSessionEnd(
    useCallback(
      () =>
        navigate(`/login?next=${encodeURIComponent(window.location.pathname + window.location.search)}`, {
          replace: true,
        }),
      [navigate],
    ),
  )

  if (me.isLoading || (signedOut && setup.isLoading)) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Spinner className="size-6" />
      </div>
    )
  }
  if (!auth && me.error && !signedOut) {
    return (
      <div className="flex min-h-svh items-center justify-center p-4">
        <ErrorState error={me.error} onRetry={() => void me.refetch()} />
      </div>
    )
  }
  if (!auth && setup.data?.required) return <Navigate to="/setup" replace />
  if (!auth) return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />
  if (auth.user.mustChangePassword && location.pathname !== "/change-password")
    return <Navigate to="/change-password" replace />

  return (
    <AuthContext.Provider value={auth}>
      <Outlet />
    </AuthContext.Provider>
  )
}

/** Only administrators may see the nested routes. */
export function RequireAdmin() {
  const { isAdmin } = useAuth()
  return isAdmin ? <Outlet /> : <Navigate to="/" replace />
}
