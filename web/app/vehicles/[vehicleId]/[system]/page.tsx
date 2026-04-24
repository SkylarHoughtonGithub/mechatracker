'use client'

import { use, useState, useEffect } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useGarage, useVehicleRepairs } from '@/lib/queries'
import { useApi, type Repair } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'

const STATUS_LABEL: Record<string, string> = {
  draft:         'Draft',
  in_progress:   'In progress',
  pending_parts: 'Pending parts',
  completed:     'Completed',
}

const STATUS_STYLE: Record<string, string> = {
  draft:         'bg-zinc-800 text-zinc-400',
  in_progress:   'bg-blue-950 text-blue-400',
  pending_parts: 'bg-amber-950 text-amber-400',
  completed:     'bg-green-950 text-green-400',
}

export default function SystemRepairsPage({ params }: { params: Promise<{ vehicleId: string; system: string }> }) {
  const { vehicleId, system } = use(params)
  const { data: garage, isLoading: garageLoading } = useGarage()
  const { data: repairs, isLoading: repairsLoading, error } = useVehicleRepairs(vehicleId, system)
  const [modalOpen, setModalOpen] = useState(false)

  const isLoading = garageLoading || repairsLoading

  useEffect(() => {
    const v = garage?.vehicles?.find(v => v.id === vehicleId)
    const sLabel = v?.systems?.find(s => s.id === system)?.label ?? system
    const vLabel = v ? `${v.year} ${v.make} ${v.model}${v.trim ? ` ${v.trim}` : ''}` : null
    if (vLabel) document.title = `${sLabel} · ${vLabel}`
  }, [garage, vehicleId, system])

  if (isLoading) return <div className="p-8 text-sm text-zinc-500">Loading...</div>
  if (error) return <div className="p-8 text-sm text-red-500">Error: {error.message}</div>

  const vehicle = garage?.vehicles?.find(v => v.id === vehicleId)
  const systemInfo = vehicle?.systems?.find(s => s.id === system)
  const vehicleLabel = vehicle ? `${vehicle.year} ${vehicle.make} ${vehicle.model}${vehicle.trim ? ` ${vehicle.trim}` : ''}` : '...'
  const systemLabel = systemInfo?.label ?? system

  return (
    <div className="p-8">
      <div className="max-w-2xl mx-auto">
        <nav className="flex items-center gap-1.5 text-xs text-zinc-500 mb-6 flex-wrap">
          <Link href="/" className="hover:text-zinc-300 transition-colors">{garage?.org_name ?? 'Garage'}</Link>
          <Chevron />
          <Link href={`/vehicles/${vehicleId}`} className="hover:text-zinc-300 transition-colors">{vehicleLabel}</Link>
          <Chevron />
          <span className="text-zinc-200">{systemLabel}</span>
        </nav>

        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-xl font-semibold text-zinc-100">{systemLabel}</h1>
            <div className="text-sm text-zinc-500 mt-0.5">{vehicleLabel}</div>
          </div>
          <button onClick={() => setModalOpen(true)} className="text-sm bg-zinc-800 border border-zinc-700 text-zinc-100 px-3 py-1.5 rounded-lg font-medium hover:bg-zinc-700 transition-colors">
            + Add repair
          </button>
        </div>

        {repairs?.length === 0 ? (
          <div className="text-sm text-zinc-600 py-8 text-center bg-zinc-900 rounded-xl border border-zinc-800">
            No repairs logged for this system yet.
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {repairs?.map(repair => <RepairRow key={repair.id} repair={repair} vehicleId={vehicleId} system={system} />)}
          </div>
        )}
      </div>

      {modalOpen && <AddRepairModal vehicleId={vehicleId} system={system} systemLabel={systemLabel} onClose={() => setModalOpen(false)} />}
    </div>
  )
}

function AddRepairModal({ vehicleId, system, systemLabel, onClose }: { vehicleId: string; system: string; systemLabel: string; onClose: () => void }) {
  const api = useApi()
  const qc = useQueryClient()
  const router = useRouter()

  const [title, setTitle]   = useState('')
  const [saving, setSaving] = useState(false)
  const [err, setErr]       = useState('')

  async function submit() {
    if (!title.trim()) { setErr('A title is required.'); return }
    setSaving(true); setErr('')
    try {
      const repair = await api.repairs.create({ vehicle_id: vehicleId, title: title.trim(), system })
      await qc.invalidateQueries({ queryKey: ['vehicle-repairs', vehicleId, system] })
      await qc.invalidateQueries({ queryKey: ['garage'] })
      onClose()
      router.push(`/vehicles/${vehicleId}/${system}/${repair.id}`)
    } catch (e: any) { setErr(e.message ?? 'Something went wrong.'); setSaving(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl shadow-2xl p-5">
        <div className="flex items-center justify-between mb-4">
          <span className="text-base font-semibold text-zinc-100">Add repair</span>
          <button onClick={onClose} className="text-zinc-600 hover:text-zinc-300 transition-colors">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-zinc-500 mb-1">System</label>
            <div className="bg-zinc-800/50 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-500">{systemLabel}</div>
          </div>
          <div>
            <label className="block text-xs text-zinc-500 mb-1">Title <span className="text-red-400">*</span></label>
            <input type="text" value={title} onChange={e => setTitle(e.target.value)} placeholder="e.g. Brake pad replacement" autoFocus
              onKeyDown={e => { if (e.key === 'Enter') submit() }}
              className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          {err && <p className="text-xs text-red-400">{err}</p>}
          <div className="flex gap-2 pt-1">
            <button onClick={onClose} className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 font-medium text-sm py-2.5 rounded-xl transition-colors">Cancel</button>
            <button onClick={submit} disabled={saving} className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors">
              {saving ? 'Creating…' : 'Create repair'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function RepairRow({ repair, vehicleId, system }: { repair: Repair; vehicleId: string; system: string }) {
  const status = repair.status ?? 'draft'
  const progress = repair.progress_pct ?? 0
  return (
    <Link href={`/vehicles/${vehicleId}/${system}/${repair.id}`} className="bg-zinc-900 rounded-xl border border-zinc-800 p-4 hover:border-zinc-700 hover:bg-zinc-800/50 transition-colors block">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm text-zinc-100">{repair.title}</div>
          {repair.started_at && <div className="text-xs text-zinc-600 mt-0.5">Started {new Date(repair.started_at).toLocaleDateString()}</div>}
          {repair.completed_at && <div className="text-xs text-zinc-600 mt-0.5">Completed {new Date(repair.completed_at).toLocaleDateString()}</div>}
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_STYLE[status] ?? STATUS_STYLE.draft}`}>{STATUS_LABEL[status] ?? status}</span>
          <svg className="w-4 h-4 text-zinc-700" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
        </div>
      </div>
      {status === 'in_progress' && progress > 0 && (
        <div className="mt-3">
          <div className="flex justify-between text-xs text-zinc-600 mb-1"><span>Progress</span><span>{progress}%</span></div>
          <div className="h-1 bg-zinc-800 rounded-full overflow-hidden">
            <div className="h-full bg-blue-500 rounded-full transition-all" style={{ width: `${progress}%` }} />
          </div>
        </div>
      )}
      {(repair.total_parts_cost ?? 0) > 0 && (
        <div className="mt-2 text-xs text-zinc-600">Parts: <span className="text-zinc-400 font-medium">${repair.total_parts_cost}</span></div>
      )}
    </Link>
  )
}

function Chevron() {
  return (
    <svg className="w-3 h-3 text-zinc-700 flex-shrink-0" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
    </svg>
  )
}