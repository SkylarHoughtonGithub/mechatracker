'use client'

import { useRouter } from 'next/navigation'
import { use, useEffect } from 'react'

// Thin convenience route: /accept-invite/<token> → /onboarding?token=<token>.
// Lets an email/share link work without baking the UI into both pages.
export default function AcceptInvitePage({
  params,
}: {
  params: Promise<{ token: string }>
}) {
  const { token } = use(params)
  const router = useRouter()
  useEffect(() => {
    router.replace(`/onboarding?token=${encodeURIComponent(token)}`)
  }, [token, router])
  return null
}
