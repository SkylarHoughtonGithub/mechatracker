'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useGarage } from '@/lib/queries'

const ALL_SYSTEMS: { id: string; label: string; icon: string }[] = [
  { id: 'engine',       label: 'Engine',         icon: '⚙️'  },
  { id: 'transmission', label: 'Transmission',    icon: '🔧'  },
  { id: 'brakes',       label: 'Brakes',          icon: '🛑'  },
  { id: 'suspension',   label: 'Suspension',      icon: '🔩'  },
  { id: 'steering',     label: 'Steering',        icon: '🎯'  },
  { id: 'electrical',   label: 'Electrical',      icon: '⚡'  },
  { id: 'cooling',      label: 'Cooling',         icon: '🌡️' },
  { id: 'fuel',         label: 'Fuel System',     icon: '⛽'  },
  { id: 'exhaust',      label: 'Exhaust',         icon: '💨'  },
  { id: 'hvac',         label: 'HVAC',            icon: '❄️'  },
  { id: 'tires',        label: 'Tires & Wheels',  icon: '🔄'  },
  { id: 'body',         label: 'Body & Interior', icon: '🚗'  },
]

export default function VehiclePage() {
  const params = useParams()
  const vehicleId = params.vehicleId as string
  const router = useRouter()

  const { data: garage, isLoading, error } = useGarage()
  const [customModalOpen, setCustomModalOpen] = useState(false)
  const [customSystem, setCustomSystem] = useState('')

  useEffect(() => {
    const v = garage?.vehicles?.find(v => v.id === vehicleId)
    if (v && garage?.org_name) {
      const lbl = `${v.year} ${v.make} ${v.model}${v.trim ? ` · ${v.trim}` : ''}`
      document.title = `${lbl} · ${garage.org_name}`
    }
  }, [garage, vehicleId])

  if (isLoading) return <div className="p-8 text-sm text-zinc-500">Loading...</div>
  if (error) return <div className="p-8 text-sm text-red-500">Error: {(error as Error).message}</div>

  const vehicle = garage?.vehicles?.find(v => v.id === vehicleId)
  if (!vehicle) return <div className="p-8 text-sm text-zinc-500">Vehicle not found.</div>

  const label = `${vehicle.year} ${vehicle.make} ${vehicle.model}${vehicle.trim ? ` · ${vehicle.trim}` : ''}`

  const systemStats: Record<string, { repair_count: number; has_open: boolean }> = {}
  for (const s of vehicle.systems ?? []) {
    if (s.id) systemStats[s.id] = { repair_count: s.repair_count ?? 0, has_open: s.has_open ?? false }
  }

  const knownIds = new Set(ALL_SYSTEMS.map(s => s.id))
  const customSystems = (vehicle.systems ?? [])
    .filter(s => s.id && !knownIds.has(s.id))
    .map(s => ({ id: s.id!, label: s.label ?? s.id!, icon: '🔨' }))

  const allDisplayed = [...ALL_SYSTEMS, ...customSystems]

  function goToCustomSystem() {
    const id = customSystem.trim().toLowerCase().replace(/\s+/g, '_')
    if (!id) return
    setCustomModalOpen(false)
    setCustomSystem('')
    router.push(`/vehicles/${vehicleId}/${id}`)
  }

  return (
    <div className="p-8">
      <div className="max-w-2xl mx-auto">
        <nav className="flex items-center gap-1.5 text-xs text-zinc-500 mb-6">
          <Link href="/" className="hover:text-zinc-300 transition-colors">{garage?.org_name ?? 'Garage'}</Link>
          <svg className="w-3 h-3 text-zinc-700 flex-shrink-0" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
          </svg>
          <span className="text-zinc-200">{label}</span>
        </nav>

        <div className="bg-zinc-900 rounded-xl border border-zinc-800 p-4 mb-6">
          <div className="font-semibold text-lg text-zinc-100">{label}</div>
          {vehicle.mileage != null && (
            <div className="text-sm text-zinc-500 mt-0.5">{vehicle.mileage.toLocaleString()} mi</div>
          )}
          {(vehicle.open_repairs ?? 0) > 0 && (
            <div className="mt-3">
              <span className="bg-blue-950 text-blue-400 text-xs px-2 py-0.5 rounded-full font-medium">
                {vehicle.open_repairs} open repair{vehicle.open_repairs !== 1 ? 's' : ''}
              </span>
            </div>
          )}
        </div>

        <div className="flex items-center justify-between mb-3">
          <h2 className="text-xs font-medium text-zinc-500 uppercase tracking-wide">Systems</h2>
          <div className="flex items-center gap-3">
            <Link href={`/vehicles/${vehicleId}/history`} className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors">
              View history →
            </Link>
            <button onClick={() => setCustomModalOpen(true)} className="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors">
              + Add system
            </button>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          {allDisplayed.map(system => {
            const stats = systemStats[system.id]
            const hasRepairs = (stats?.repair_count ?? 0) > 0
            return (
              <Link
                key={system.id}
                href={`/vehicles/${vehicleId}/${system.id}`}
                className="flex items-center gap-3 bg-zinc-900 rounded-xl border border-zinc-800 p-4 hover:border-zinc-700 hover:bg-zinc-800/50 transition-colors"
              >
                <div className="text-xl w-7 text-center flex-shrink-0">{system.icon}</div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-sm text-zinc-100">{system.label}</div>
                  <div className="text-xs mt-0.5">
                    {hasRepairs
                      ? <span className="text-zinc-500">{stats.repair_count} repair{stats.repair_count !== 1 ? 's' : ''}</span>
                      : <span className="text-zinc-700">No repairs yet</span>
                    }
                  </div>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {stats?.has_open && (
                    <span className="text-xs bg-blue-950 text-blue-400 px-2 py-0.5 rounded-full font-medium">open</span>
                  )}
                  <svg className="w-4 h-4 text-zinc-700" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                  </svg>
                </div>
              </Link>
            )
          })}
        </div>
      </div>

    {customModalOpen && (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setCustomModalOpen(false)} />
        <div className="relative w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl shadow-2xl p-5">
          <div className="flex items-center justify-between mb-4">
            <span className="text-base font-semibold text-zinc-100">Add system</span>
            <button onClick={() => setCustomModalOpen(false)} className="text-zinc-600 hover:text-zinc-300 transition-colors">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <label className="block text-xs text-zinc-500 mb-1.5">System name</label>
          <input
            type="text"
            value={customSystem}
            onChange={e => setCustomSystem(e.target.value)}
            placeholder="e.g. Winch, Lift kit, Roof rack…"
            autoFocus
            onKeyDown={e => { if (e.key === 'Enter') goToCustomSystem() }}
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="text-xs text-zinc-600 mt-2 mb-5">You'll land on the repair list for this system.</p>
          <div className="flex gap-2">
            <button onClick={() => setCustomModalOpen(false)} className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 font-medium text-sm py-2.5 rounded-xl transition-colors">
              Cancel
            </button>
            <button onClick={goToCustomSystem} disabled={!customSystem.trim()} className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors">
              Go to system
            </button>
          </div>
        </div>
      </div>
    )}
    </div>
  )
}