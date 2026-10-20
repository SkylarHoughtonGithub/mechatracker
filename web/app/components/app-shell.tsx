'use client'

import { usePathname } from 'next/navigation'
import { useAuth } from '../../lib/auth-provider'
import { AppHeader } from './app-header'
import { Sidebar } from './sidebar'

// Routes rendered without the signed-in chrome. The middleware lets
// /sign-in and /accept-invite through without a cookie; /onboarding needs
// a cookie but no users row yet (AuthProvider.status === 'needsOnboarding').
const CHROMELESS_PREFIXES = ['/sign-in', '/onboarding', '/accept-invite']

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const { status } = useAuth()

  const isChromeless = CHROMELESS_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`),
  )

  if (isChromeless) {
    return <>{children}</>
  }

  if (status === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center text-sm text-zinc-500">
        Loading…
      </div>
    )
  }

  // signedOut / needsOnboarding on a chromed route: the middleware (for
  // signedOut) and the AuthProvider effect in /onboarding (for needsOnboarding)
  // redirect. Render nothing in the meantime to avoid a flash of chrome.
  if (status !== 'signedIn') {
    return null
  }

  return (
    <>
      <AppHeader />
      <div className="flex min-h-[calc(100vh-3rem)]">
        <Sidebar />
        <main className="flex-1 min-w-0">{children}</main>
      </div>
    </>
  )
}
