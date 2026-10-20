'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useRef, useState } from 'react'
import { useAuth } from '../../lib/auth-provider'

export function UserMenu() {
  const router = useRouter()
  const { user, org, google, signOut } = useAuth()
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  // Click-outside to close.
  useEffect(() => {
    if (!open) return
    const onDocClick = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open])

  if (!user) return null

  const display = user.name || user.email
  const picture = user.picture_url || google?.picture_url
  const initials = (user.name || user.email || '?')
    .split(/\s+/)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase() ?? '')
    .join('')

  const handleSignOut = async () => {
    await signOut()
    router.replace('/sign-in')
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex items-center justify-center w-8 h-8 rounded-full bg-zinc-800 text-zinc-100 text-xs font-medium hover:bg-zinc-700 focus:outline-none focus:ring-2 focus:ring-zinc-600 overflow-hidden"
      >
        {picture ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={picture} alt="" className="w-full h-full object-cover" />
        ) : (
          <span>{initials || '?'}</span>
        )}
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-10 w-64 rounded-md border border-zinc-800 bg-zinc-950 shadow-lg text-sm overflow-hidden"
        >
          <div className="px-3 py-3 border-b border-zinc-800">
            <div className="text-zinc-100 truncate">{display}</div>
            <div className="text-zinc-400 text-xs truncate">{user.email}</div>
            {org && (
              <div className="text-zinc-500 text-xs mt-1 truncate">
                {org.name} · {user.role}
              </div>
            )}
          </div>
          <button
            type="button"
            onClick={handleSignOut}
            className="w-full text-left px-3 py-2 text-zinc-300 hover:bg-zinc-900 hover:text-zinc-100"
          >
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}
