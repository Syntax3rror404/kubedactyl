import { createContext, useContext } from "react"

import type { UserView } from "@/lib/types"

export interface AuthState {
  user: UserView
  isAdmin: boolean
}

/** Provided by RequireAuth (features/auth/components/require-auth.tsx); read it with useAuth(). */
export const AuthContext = createContext<AuthState | null>(null)

/** The signed-in user; only available below <RequireAuth>. */
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within RequireAuth")
  return ctx
}
