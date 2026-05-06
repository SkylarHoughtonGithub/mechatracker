'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { useGarage } from '@/lib/queries'
import { useApi } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'
import { DarkModal, CancelButton } from './components/modal'
import { Stat } from './components/stat'
import { PageSpinner } from './components/spinner'

export default function HomePage() {
  const { data, isLoading, error } = useGarage()
  const [modalOpen, setModalOpen] = useState(false)

  useEffect(() => {
    if (data?.org_name) document.title = data.org_name
  }, [data?.org_name])

  if (isLoading) return <PageSpinner />
  if (error) return (
    <div className="p-8 text-sm text-red-500 text-center">
      <p>Failed to load garage.</p>
      <p className="text-zinc-600 mt-1">{error.message}</p>
    </div>
  )

  return (
    <div className="px-4 py-6 sm:p-8">
      <div className="max-w-2xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-semibold text-zinc-100">{data?.org_name ?? 'Garage'}</h1>
          <button
            onClick={() => setModalOpen(true)}
            className="text-sm bg-zinc-800 border border-zinc-700 text-zinc-100 px-3 py-1.5 rounded-lg font-medium hover:bg-zinc-700 transition-colors"
          >
            + Add vehicle
          </button>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-8">
          <Stat label="Vehicles"      value={data?.stats?.total_vehicles ?? 0} />
          <Stat label="Open repairs"  value={data?.stats?.open_repairs ?? 0} />
          <Stat label="Repairs (30d)" value={data?.stats?.repairs_30d ?? 0} />
          <Stat label="Parts spend"   value={`$${data?.stats?.parts_spend_30d ?? 0}`} />
        </div>

        {(!data?.vehicles || data.vehicles.length === 0) ? (
          <div className="rounded-xl border border-zinc-800 p-12 text-center">
            <div className="text-3xl mb-3">🔧</div>
            <div className="text-sm font-medium text-zinc-300 mb-1">No vehicles yet</div>
            <div className="text-xs text-zinc-600 mb-5">Add your first vehicle to start tracking repairs.</div>
            <button
              onClick={() => setModalOpen(true)}
              className="text-sm bg-zinc-800 border border-zinc-700 text-zinc-100 px-4 py-2 rounded-lg font-medium hover:bg-zinc-700 transition-colors"
            >
              + Add vehicle
            </button>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {data.vehicles.map(vehicle => (
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
        )}
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
    } catch (e) { setErr(e instanceof Error ? e.message : 'Something went wrong.') }
    finally { setSaving(false) }
  }

  return (
    <DarkModal title="Add vehicle" onClose={onClose}>
      <div className="space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
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
          <CancelButton onClick={onClose} />
          <button onClick={submit} disabled={saving} className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors">
            {saving ? 'Adding…' : 'Add vehicle'}
          </button>
        </div>
      </div>
    </DarkModal>
  )
}
