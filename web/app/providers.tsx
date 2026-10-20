'use client'

import { GoogleOAuthProvider } from '@react-oauth/google'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { AuthProvider } from '../lib/auth-provider'
import { ToastProvider } from './components/toast'

const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? ''

export function Providers({ children }: { children: React.ReactNode }) {
    const [queryClient] = useState(() => new QueryClient())
    return (
        <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
            <QueryClientProvider client={queryClient}>
                <AuthProvider>
                    <ToastProvider>
                        {children}
                    </ToastProvider>
                </AuthProvider>
            </QueryClientProvider>
        </GoogleOAuthProvider>
    )
}
