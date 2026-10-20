'use client'

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { ApiError, auth, type MeResponse, type MeUser, type MeOrg, type MeGoogleClaims } from './api'

export type AuthStatus = 'loading' | 'signedOut' | 'needsOnboarding' | 'signedIn'

type AuthState = {
  status: AuthStatus
  user: MeUser | null
  org: MeOrg | null
  google: MeGoogleClaims | null
}

type AuthContextValue = AuthState & {
  // Replace the entire state atomically — used after /auth/google, /orgs,
  // and /invites/{token}/accept return a fresh /auth/me-shaped response.
  setFromMe: (me: MeResponse) => void
  // Refetch /auth/me from scratch; use when nothing is known about the
  // current session (first page load, after logout).
  refresh: () => Promise<void>
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

function statusFromMe(me: MeResponse): AuthStatus {
  if (me.user && me.org) return 'signedIn'
  if (me.needs_onboarding) return 'needsOnboarding'
  return 'signedOut'
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    status: 'loading',
    user: null,
    org: null,
    google: null,
  })
  // Guards against the refresh StrictMode double-fire + concurrent calls.
  const refreshing = useRef(false)

  const applyMe = useCallback((me: MeResponse) => {
    setState({
      status: statusFromMe(me),
      user: me.user ?? null,
      org: me.org ?? null,
      google: me.google ?? null,
    })
  }, [])

  const refresh = useCallback(async () => {
    if (refreshing.current) return
    refreshing.current = true
    try {
      const me = await auth.me()
      applyMe(me)
    } catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        setState({ status: 'signedOut', user: null, org: null, google: null })
      } else {
        // Network or 5xx — surface as signedOut rather than hanging in loading;
        // the user can retry via sign-in.
        setState({ status: 'signedOut', user: null, org: null, google: null })
      }
    } finally {
      refreshing.current = false
    }
  }, [applyMe])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const signOut = useCallback(async () => {
    try {
      await auth.logout()
    } catch {
      // Even if logout fails on the server, clear local state so the user
      // isn't trapped. The next request will 401 and route to /sign-in.
    }
    setState({ status: 'signedOut', user: null, org: null, google: null })
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({ ...state, setFromMe: applyMe, refresh, signOut }),
    [state, applyMe, refresh, signOut],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}
