'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useGarage } from '@/lib/queries'
import { useApi } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'
import { UserButton } from '@clerk/nextjs'

export default function HomePage() {
  const { data, isLoading, error } = useGarage()
  const [modalOpen, setModalOpen] = useState(false)

  if (isLoading) return <div className="p-8 text-sm text-zinc-500">Loading garage...</div>
  if (error) return <div className="p-8 text-sm text-red-500">Error: {error.message}</div>

  return (
    <div className="p-8">
      <div className="max-w-2xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-semibold text-zinc-100">Garage</h1>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setModalOpen(true)}
              className="text-sm bg-zinc-800 border border-zinc-700 text-zinc-100 px-3 py-1.5 rounded-lg font-medium hover:bg-zinc-700 transition-colors"
            >
              + Add vehicle
            </button>
            <UserButton />
          </div>
        </div>

        <div className="grid grid-cols-4 gap-3 mb-8">
          <Stat label="Vehicles"      value={data?.stats?.total_vehicles ?? 0} />
          <Stat label="Open repairs"  value={data?.stats?.open_repairs ?? 0} />
          <Stat label="Repairs (30d)" value={data?.stats?.repairs_30d ?? 0} />
          <Stat label="Parts spend"   value={`$${data?.stats?.parts_spend_30d ?? 0}`} />
        </div>

        <div className="flex flex-col gap-2">
          {data?.vehicles?.map(vehicle => (
            <Link
              key={vehicle.id}
              href={`/vehicles/${vehicle.id}`}
              className="flex items-center justify-between bg-zinc-900 rounded-xl border border-zinc-800 p-4 hover:border-zinc-700 hover:bg-zinc-800/50 transition-colors"
            >
              <div>
                <div className="font-medium text-sm text-zinc-100">
                  {vehicle.year} {vehicle.make} {vehicle.model}
                  {vehicle.trim && <span className="text-zinc-500"> · {vehicle.trim}</span>}
                </div>
                {vehicle.mileage && (
                  <div className="text-xs text-zinc-600 mt-0.5">{vehicle.mileage.toLocaleString()} mi</div>
                )}
              </div>
              <div className="flex items-center gap-3 text-xs">
                {(vehicle.open_repairs ?? 0) > 0 && (
                  <span className="bg-blue-950 text-blue-400 px-2 py-0.5 rounded-full font-medium">
                    {vehicle.open_repairs} open
                  </span>
                )}
                {(vehicle.systems?.length ?? 0) > 0 && (
                  <span className="text-zinc-600">
                    {vehicle.systems?.length} system{vehicle.systems?.length !== 1 ? 's' : ''}
                  </span>
                )}
                <svg className="w-4 h-4 text-zinc-700" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                </svg>
              </div>
            </Link>
          ))}
        </div>
      </div>

      {modalOpen && <AddVehicleModal onClose={() => setModalOpen(false)} />}
    </div>
  )
}

function AddVehicleModal({ onClose }: { onClose: () => void }) {
  const api = useApi()
  const qc = useQueryClient()

  const currentYear = new Date().getFullYear()
  const [year, setYear]       = useState(String(currentYear))
  const [make, setMake]       = useState('')
  const [model, setModel]     = useState('')
  const [trim, setTrim]       = useState('')
  const [mileage, setMileage] = useState('')
  const [saving, setSaving]   = useState(false)
  const [err, setErr]         = useState('')

  async function submit() {
    const yearNum = parseInt(year, 10)
    if (!make.trim() || !model.trim() || isNaN(yearNum)) { setErr('Year, make, and model are required.'); return }
    setSaving(true); setErr('')
    try {
      await api.vehicles.create({ year: yearNum, make: make.trim(), model: model.trim(), trim: trim.trim() || undefined, mileage: mileage ? parseInt(mileage, 10) : undefined })
      await qc.invalidateQueries({ queryKey: ['garage'] })
      onClose()
    } catch (e: any) { setErr(e.message ?? 'Something went wrong.') }
    finally { setSaving(false) }
  }

  return (
    <DarkBottomModal onClose={onClose}>
      <div className="flex items-center justify-between mb-4">
        <span className="text-base font-semibold text-zinc-100">Add vehicle</span>
        <button onClick={onClose} className="text-zinc-600 hover:text-zinc-300 transition-colors">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      <div className="space-y-3">
        <div className="grid grid-cols-3 gap-2">
          {[
            { label: 'Year *', value: year, set: setYear, placeholder: '2019', type: 'number' },
            { label: 'Make *', value: make, set: setMake, placeholder: 'Ford',  type: 'text'   },
            { label: 'Model *', value: model, set: setModel, placeholder: 'F-150', type: 'text' },
          ].map(f => (
            <div key={f.label}>
              <label className="block text-xs text-zinc-500 mb-1">{f.label}</label>
              <input type={f.type} value={f.value} onChange={e => f.set(e.target.value)} placeholder={f.placeholder}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
          ))}
        </div>
        <div className="grid grid-cols-2 gap-2">
          {[
            { label: 'Trim', value: trim, set: setTrim, placeholder: 'XLT' },
            { label: 'Mileage', value: mileage, set: setMileage, placeholder: '84200' },
          ].map(f => (
            <div key={f.label}>
              <label className="block text-xs text-zinc-500 mb-1">{f.label}</label>
              <input type="text" value={f.value} onChange={e => f.set(e.target.value)} placeholder={f.placeholder}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
          ))}
        </div>
        {err && <p className="text-xs text-red-400">{err}</p>}
        <div className="flex gap-2 pt-1">
          <button onClick={onClose} className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 font-medium text-sm py-2.5 rounded-xl transition-colors">Cancel</button>
          <button onClick={submit} disabled={saving} className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors">
            {saving ? 'Adding…' : 'Add vehicle'}
          </button>
        </div>
      </div>
    </DarkBottomModal>
  )
}

function DarkBottomModal({ onClose, children }: { onClose: () => void; children: React.ReactNode }) {
  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 50, display: 'flex', flexDirection: 'column', justifyContent: 'flex-end' }}>
      <div style={{ position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)' }} onClick={onClose} />
      <div style={{ position: 'relative', background: '#18181b', borderTop: '1px solid #27272a', borderRadius: '16px 16px 0 0', padding: '20px', maxHeight: '80vh', overflowY: 'auto' }}>
        {children}
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-zinc-900 rounded-xl border border-zinc-800 p-3">
      <div className="text-xs text-zinc-500 mb-1">{label}</div>
      <div className="text-xl font-semibold text-zinc-100">{value}</div>
    </div>
  )
}