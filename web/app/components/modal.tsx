'use client'

import { useEffect, useState } from 'react'

export function DarkModal({
  title,
  onClose,
  children,
}: {
  title?: string
  onClose: () => void
  children: React.ReactNode
}) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl shadow-2xl p-5">
        {title && (
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-zinc-100">{title}</h2>
            <CloseButton onClick={onClose} />
          </div>
        )}
        {children}
      </div>
    </div>
  )
}

export function CloseButton({ onClick }: { onClick: () => void }) {
  return (
    <button onClick={onClick} className="text-zinc-600 hover:text-zinc-300 transition-colors" aria-label="Close">
      <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  )
}

export function CancelButton({ onClick, disabled }: { onClick: () => void; disabled?: boolean }) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 disabled:opacity-40 font-medium text-sm py-2.5 rounded-xl transition-colors"
    >
      Cancel
    </button>
  )
}

export function ConfirmModal({
  title,
  message,
  confirmLabel,
  danger,
  onConfirm,
  onClose,
}: {
  title: string
  message: string
  confirmLabel: string
  danger?: boolean
  onConfirm: () => Promise<void>
  onClose: () => void
}) {
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  async function handleConfirm() {
    setLoading(true); setErr('')
    try { await onConfirm() }
    catch (e) { setErr(e instanceof Error ? e.message : 'Something went wrong.'); setLoading(false) }
  }

  return (
    <DarkModal title={title} onClose={onClose}>
      <p className="text-sm text-zinc-400 mb-5">{message}</p>
      {err && <p className="text-xs text-red-400 mb-3">{err}</p>}
      <div className="flex gap-2">
        <CancelButton onClick={onClose} disabled={loading} />
        <button
          onClick={handleConfirm}
          disabled={loading}
          className={`flex-1 font-medium text-sm py-2.5 rounded-xl transition-colors disabled:opacity-40 ${
            danger ? 'bg-red-600 hover:bg-red-500 text-white' : 'bg-zinc-100 hover:bg-white text-zinc-900'
          }`}
        >
          {loading ? 'Deleting…' : confirmLabel}
        </button>
      </div>
    </DarkModal>
  )
}

export function ModalActions({
  onClose,
  onSubmit,
  saving,
  label,
}: {
  onClose: () => void
  onSubmit: () => void
  saving: boolean
  label: string
}) {
  return (
    <div className="flex gap-2 pt-1">
      <CancelButton onClick={onClose} />
      <button
        onClick={onSubmit}
        disabled={saving}
        className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors"
      >
        {saving ? 'Saving…' : label}
      </button>
    </div>
  )
}
