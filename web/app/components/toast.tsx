'use client'

import { createContext, useContext, useState, useCallback } from 'react'

type ToastType = 'success' | 'error' | 'info'
type Toast = { id: string; message: string; type: ToastType }
type ToastCtx = { toast: (message: string, type?: ToastType) => void }

const Ctx = createContext<ToastCtx>({ toast: () => {} })

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const toast = useCallback((message: string, type: ToastType = 'info') => {
    const id = Math.random().toString(36).slice(2)
    setToasts(prev => [...prev, { id, message, type }])
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 4000)
  }, [])

  return (
    <Ctx.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-4 left-1/2 -translate-x-1/2 flex flex-col items-center gap-2 z-[60] pointer-events-none">
        {toasts.map(t => (
          <div
            key={t.id}
            className={`px-4 py-3 rounded-xl text-sm font-medium shadow-xl border whitespace-nowrap pointer-events-auto
              ${t.type === 'error' ? 'bg-red-950 border-red-800 text-red-300' :
                t.type === 'success' ? 'bg-green-950 border-green-800 text-green-300' :
                'bg-zinc-800 border-zinc-700 text-zinc-100'}`}
          >
            {t.message}
          </div>
        ))}
      </div>
    </Ctx.Provider>
  )
}

export function useToast() { return useContext(Ctx) }
