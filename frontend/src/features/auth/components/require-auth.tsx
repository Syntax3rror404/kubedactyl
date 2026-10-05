import { useCallback } from "react"
import { Navigate, Outlet, useLocation, useNavigate } from "react-router"

import { Spinner } from "@/components/ui/spinner"
import { AuthContext, useAuth } from "@/hooks/use-auth"
import { useMe, useSessionEnd, useSetupStatus } from "@/lib/queries"

/**
 * Loads the current user; without a session the visitor is sent to /login (or /setup on the first start). A user
 * whose password an administrator set chooses an own one first (/change-password).
 */
export function RequireAuth() {
  const me = useMe()
  const setup = useSetupStatus(!me.isLoading && !me.data)
  const navigate = useNavigate()
  const location = useLocation()
  const next = location.pathname + location.search

  useSessionEnd(
    useCallback(
      () =>
        navigate(`/login?next=${encodeURIComponent(window.location.pathname + window.location.search)}`, {
          replace: true,
        }),
      [navigate],
    ),
  )

  if (me.isLoading || (!me.data && setup.isLoading)) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Spinner className="size-6" />
      </div>
    )
  }
  if (!me.data && setup.data?.required) return <Navigate to="/setup" replace />
  if (!me.data) return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />
  if (me.data.mustChangePassword && location.pathname !== "/change-password")
    return <Navigate to="/change-password" replace />

  return (
    <AuthContext.Provider value={{ user: me.data, isAdmin: me.data.role === "admin" }}>
      <Outlet />
    </AuthContext.Provider>
  )
}

/** Only administrators may see the nested routes. */
export function RequireAdmin() {
  const { isAdmin } = useAuth()
  return isAdmin ? <Outlet /> : <Navigate to="/" replace />
}
