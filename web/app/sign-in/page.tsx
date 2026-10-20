'use client'

import { GoogleLogin, type CredentialResponse } from '@react-oauth/google'
import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useState } from 'react'
import { auth } from '../../lib/api'
import { useAuth } from '../../lib/auth-provider'

export default function SignInPage() {
  return (
    <Suspense fallback={<SignInShell />}>
      <SignInBody />
    </Suspense>
  )
}

// The outer shell renders during Suspense fallback so the layout doesn't
// pop in; the Google button only hydrates once the client reads searchParams.
function SignInShell({ children }: { children?: React.ReactNode } = {}) {
  return (
    <div className="min-h-[calc(100vh-3rem)] flex items-center justify-center px-6">
      <div className="w-full max-w-sm space-y-6 text-center">
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-zinc-100">Sign in to mechatracker</h1>
          <p className="text-sm text-zinc-400">
            Use your Google account. New here? You&apos;ll pick or create a workspace next.
          </p>
        </div>
        {children}
      </div>
    </div>
  )
}

function SignInBody() {
  const router = useRouter()
  const search = useSearchParams()
  const { status, setFromMe } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const next = search.get('next') || '/'

  useEffect(() => {
    if (status === 'signedIn') router.replace(next)
    else if (status === 'needsOnboarding') router.replace('/onboarding')
  }, [status, router, next])

  const handleCredential = async (credential: CredentialResponse) => {
    const idToken = credential.credential
    if (!idToken) {
      setError('Google did not return a credential. Try again.')
      return
    }
    setPending(true)
    setError(null)
    try {
      const me = await auth.google(idToken)
      setFromMe(me)
      if (me.user && me.org) router.replace(next)
      else router.replace('/onboarding')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed.')
    } finally {
      setPending(false)
    }
  }

  return (
    <SignInShell>
      <div className="flex justify-center">
        <GoogleLogin
          onSuccess={handleCredential}
          onError={() => setError('Google sign-in failed.')}
          theme="filled_black"
          size="large"
          text="signin_with"
          shape="pill"
        />
      </div>

      {pending && <p className="text-sm text-zinc-500">Signing in…</p>}
      {error && (
        <p className="text-sm text-red-400 border border-red-800/40 bg-red-900/20 rounded-md px-3 py-2">
          {error}
        </p>
      )}
    </SignInShell>
  )
}
