'use client'

import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useState } from 'react'
import { onboarding } from '../../lib/api'
import { useAuth } from '../../lib/auth-provider'

type Mode = 'create' | 'join'

export default function OnboardingPage() {
  return (
    <Suspense fallback={null}>
      <OnboardingBody />
    </Suspense>
  )
}

function OnboardingBody() {
  const router = useRouter()
  const search = useSearchParams()
  const { status, google, setFromMe } = useAuth()

  const prefilledToken = search.get('token') ?? ''
  const [mode, setMode] = useState<Mode>(prefilledToken ? 'join' : 'create')
  const [orgName, setOrgName] = useState('')
  const [inviteToken, setInviteToken] = useState(prefilledToken)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  useEffect(() => {
    if (status === 'signedOut') router.replace('/sign-in')
    else if (status === 'signedIn') router.replace('/')
  }, [status, router])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      let me
      if (mode === 'create') {
        if (!orgName.trim()) throw new Error('Workspace name is required.')
        me = await onboarding.createOrg(orgName.trim())
      } else {
        if (!inviteToken.trim()) throw new Error('Invite token is required.')
        me = await onboarding.acceptInvite(inviteToken.trim())
      }
      setFromMe(me)
      router.replace('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Onboarding failed.')
    } finally {
      setPending(false)
    }
  }

  if (status !== 'needsOnboarding') {
    return null
  }

  return (
    <div className="min-h-[calc(100vh-3rem)] flex items-center justify-center px-6">
      <div className="w-full max-w-md space-y-6">
        <header className="space-y-2 text-center">
          <h1 className="text-2xl font-semibold text-zinc-100">Welcome to mechatracker</h1>
          <p className="text-sm text-zinc-400">
            {google?.email ? (
              <>Signed in as <span className="text-zinc-200">{google.email}</span>. </>
            ) : null}
            Create a workspace, or paste an invite from someone else&apos;s.
          </p>
        </header>

        <div className="flex rounded-md border border-zinc-800 overflow-hidden text-sm">
          <button
            type="button"
            onClick={() => setMode('create')}
            className={`flex-1 px-4 py-2 ${mode === 'create' ? 'bg-zinc-100 text-zinc-900' : 'bg-zinc-950 text-zinc-300 hover:bg-zinc-900'}`}
          >
            Create workspace
          </button>
          <button
            type="button"
            onClick={() => setMode('join')}
            className={`flex-1 px-4 py-2 ${mode === 'join' ? 'bg-zinc-100 text-zinc-900' : 'bg-zinc-950 text-zinc-300 hover:bg-zinc-900'}`}
          >
            Accept invite
          </button>
        </div>

        <form onSubmit={submit} className="space-y-4">
          {mode === 'create' ? (
            <label className="block">
              <span className="text-sm text-zinc-300">Workspace name</span>
              <input
                type="text"
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                placeholder="e.g. Shoutsky Garage"
                autoFocus
                className="mt-1 w-full rounded-md bg-zinc-950 border border-zinc-800 px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:ring-2 focus:ring-zinc-600"
              />
            </label>
          ) : (
            <label className="block">
              <span className="text-sm text-zinc-300">Invite token</span>
              <input
                type="text"
                value={inviteToken}
                onChange={(e) => setInviteToken(e.target.value)}
                placeholder="Paste the token from the invite link"
                autoFocus
                className="mt-1 w-full rounded-md bg-zinc-950 border border-zinc-800 px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:ring-2 focus:ring-zinc-600 font-mono"
              />
            </label>
          )}

          {error && (
            <p className="text-sm text-red-400 border border-red-800/40 bg-red-900/20 rounded-md px-3 py-2">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={pending}
            className="w-full rounded-md bg-zinc-100 text-zinc-900 px-4 py-2 text-sm font-medium hover:bg-white disabled:opacity-60"
          >
            {pending ? 'Working…' : mode === 'create' ? 'Create workspace' : 'Accept invite'}
          </button>
        </form>
      </div>
    </div>
  )
}
