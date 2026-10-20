'use client'

import Link from 'next/link'
import { UserMenu } from './user-menu'

export function AppHeader() {
  return (
    <header className="bg-zinc-950 border-b border-zinc-800 sticky top-0 z-30">
      <div className="max-w-2xl mx-auto px-6 h-12 flex items-center justify-between">
        <Link
          href="/"
          className="text-sm font-semibold text-zinc-100 hover:text-white transition-colors tracking-tight"
        >
          mechatracker
        </Link>
        <UserMenu />
      </div>
    </header>
  )
}
