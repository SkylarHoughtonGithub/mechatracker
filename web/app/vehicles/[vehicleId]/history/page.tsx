'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useGarage, useVehicleHistory } from '@/lib/queries'
import type { RepairHistory } from '@/lib/api'
import { Stat } from '@/app/components/stat'
import { Chevron } from '@/app/components/chevron'
import { PageSpinner } from '@/app/components/spinner'

const SYSTEM_LABELS: Record<string, string> = {
  engine:       'Engine',
  suspension:   'Suspension',
  brakes:       'Brakes',
  electrical:   'Electrical',
  transmission: 'Transmission',
  cooling:      'Cooling',
  fuel:         'Fuel System',
  exhaust:      'Exhaust',
  steering:     'Steering',
  tires:        'Tires & Wheels',
  hvac:         'HVAC',
  body:         'Body & Interior',
}

export default function VehicleHistoryPage() {
  const params = useParams()
  const vehicleId = params.vehicleId as string

  const { data: garage, isLoading: garageLoading } = useGarage()
  const { data: history, isLoading: historyLoading, error } = useVehicleHistory(vehicleId)

  const [activeSystem, setActiveSystem] = useState('all')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const isLoading = garageLoading || historyLoading

  useEffect(() => {
    const v = garage?.vehicles?.find(v => v.id === vehicleId)
    if (v && garage?.org_name) {
      const lbl = `${v.year} ${v.make} ${v.model}${v.trim ? ` ${v.trim}` : ''}`
      document.title = `History · ${lbl} · ${garage.org_name}`
    }
  }, [garage, vehicleId])

  if (isLoading) return <PageSpinner />
  if (error) return (
    <div className="p-8 text-sm text-red-500 text-center">
      <p>Failed to load history.</p>
      <p className="text-zinc-600 mt-1">{(error as Error).message}</p>
    </div>
  )

  const vehicle = garage?.vehicles?.find(v => v.id === vehicleId)
  const vehicleLabel = vehicle
    ? `${vehicle.year} ${vehicle.make} ${vehicle.model}${vehicle.trim ? ` ${vehicle.trim}` : ''}`
    : '...'

  const systems = ['all', ...Array.from(new Set(history?.map(h => h.system).filter((s): s is string => !!s) ?? []))]

  const filtered = activeSystem === 'all'
    ? (history ?? [])
    : (history ?? []).filter(h => h.system === activeSystem)

  const totalCost = (history ?? []).reduce((sum, h) => sum + (Number(h.total_parts_cost) ?? 0), 0)

  return (
    <main className="min-h-screen bg-black text-white">
      {/* Sticky page header — sits below the global app header (top-12) */}
      <div className="bg-zinc-950 border-b border-zinc-800 sticky top-12 z-10">
        <div className="max-w-2xl mx-auto px-6 py-4">
          <nav className="flex items-center gap-1.5 text-xs text-zinc-500 mb-3 flex-wrap">
            <Link href="/" className="hover:text-zinc-300 transition-colors">{garage?.org_name ?? 'Garage'}</Link>
            <Chevron />
            <Link href={`/vehicles/${vehicleId}`} className="hover:text-zinc-300 transition-colors">{vehicleLabel}</Link>
            <Chevron />
            <span className="text-zinc-200">History</span>
          </nav>
          <h1 className="text-xl font-semibold">Repair History</h1>
          <p className="text-sm text-zinc-500 mt-0.5">{vehicleLabel}</p>
        </div>
      </div>

      <div className="max-w-2xl mx-auto px-6 py-6 space-y-4">

        <div className="grid grid-cols-3 gap-3">
          <Stat label="Total repairs" value={history?.length ?? 0} />
          <Stat label="Parts spend" value={`$${totalCost.toFixed(0)}`} />
          <Stat label="Systems" value={systems.length - 1} />
        </div>

        {systems.length > 1 && (
          <div className="flex gap-2 flex-wrap">
            {systems.map(sys => (
              <button
                key={sys}
                onClick={() => setActiveSystem(sys)}
                className={`text-xs px-3 py-1.5 rounded-full border font-medium transition-colors
                  ${activeSystem === sys
                    ? 'bg-zinc-100 text-zinc-900 border-zinc-100'
                    : 'border-zinc-700 text-zinc-400 hover:border-zinc-500'}`}
              >
                {sys === 'all' ? 'All' : SYSTEM_LABELS[sys] ?? sys}
              </button>
            ))}
          </div>
        )}

        {filtered.length === 0 ? (
          <div className="rounded-xl border border-zinc-800 p-8 text-center text-sm text-zinc-600">
            No completed repairs{activeSystem !== 'all' ? ` for ${SYSTEM_LABELS[activeSystem] ?? activeSystem}` : ''}.
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {filtered.map(entry => (
              <HistoryCard
                key={entry.id}
                entry={entry}
                expanded={expandedId === entry.id}
                onToggle={() => setExpandedId(expandedId === entry.id ? null : entry.id!)}
              />
            ))}
          </div>
        )}

      </div>
    </main>
  )
}

function HistoryCard({
  entry,
  expanded,
  onToggle,
}: {
  entry: RepairHistory
  expanded: boolean
  onToggle: () => void
}) {
  const parts = parseSnapshot(entry.parts_snapshot)
  const notes = parseSnapshot(entry.notes_snapshot)

  return (
    <div className="bg-zinc-900 rounded-xl border border-zinc-800 overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full text-left px-4 py-4 flex items-start justify-between gap-3 hover:bg-zinc-800/50 transition-colors"
      >
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm text-zinc-100">{entry.title}</div>
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            <span className="text-xs text-zinc-500">
              {SYSTEM_LABELS[entry.system ?? ''] ?? entry.system}
            </span>
            {entry.completed_at && (
              <span className="text-xs text-zinc-600">
                {new Date(entry.completed_at).toLocaleDateString([], {
                  month: 'short', day: 'numeric', year: 'numeric',
                })}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-3 flex-shrink-0">
          {(Number(entry.total_parts_cost) ?? 0) > 0 && (
            <span className="text-sm font-medium text-zinc-300">
              ${Number(entry.total_parts_cost).toFixed(0)}
            </span>
          )}
          <svg
            className={`w-4 h-4 text-zinc-600 transition-transform ${expanded ? 'rotate-90' : ''}`}
            fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
          </svg>
        </div>
      </button>

      {expanded && (
        <div className="px-4 pb-4 border-t border-zinc-800 pt-3 space-y-3">
          <div className="grid grid-cols-2 gap-3 text-xs">
            <div>
              <div className="text-zinc-600 mb-1">Steps completed</div>
              <div className="text-zinc-300 font-medium">{entry.total_steps}</div>
            </div>
            {(entry.total_parts_cost ?? 0) > 0 && (
              <div>
                <div className="text-zinc-600 mb-1">Parts cost</div>
                <div className="text-zinc-300 font-medium">${Number(entry.total_parts_cost).toFixed(2)}</div>
              </div>
            )}
            {entry.started_at && entry.completed_at && (
              <div>
                <div className="text-zinc-600 mb-1">Duration</div>
                <div className="text-zinc-300 font-medium">
                  {getDuration(entry.started_at, entry.completed_at)}
                </div>
              </div>
            )}
          </div>

          {parts.length > 0 && (
            <div>
              <div className="text-xs text-zinc-600 mb-1.5">Parts used</div>
              <div className="space-y-1">
                {(parts as {name?: string; part_number?: string; cost?: number}[]).map((p, i) => (
                  <div key={i} className="flex justify-between text-xs">
                    <span className="text-zinc-400">
                      {p.name}{p.part_number && <span className="text-zinc-600 ml-1 font-mono">{p.part_number}</span>}
                    </span>
                    {p.cost && <span className="text-zinc-500">${Number(p.cost).toFixed(2)}</span>}
                  </div>
                ))}
              </div>
            </div>
          )}

          {notes.length > 0 && (
            <div>
              <div className="text-xs text-zinc-600 mb-1.5">Notes</div>
              <div className="space-y-1.5">
                {(notes as {body?: string; author_name?: string}[]).map((n, i) => (
                  <div key={i} className="text-xs text-zinc-400 leading-relaxed">
                    {n.body}
                    {n.author_name && <span className="text-zinc-600 ml-1">— {n.author_name}</span>}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function parseSnapshot(snapshot: unknown): unknown[] {
  if (!snapshot) return []
  if (Array.isArray(snapshot)) return snapshot
  if (typeof snapshot === 'string') {
    try {
      const decoded = atob(snapshot)
      const parsed = JSON.parse(decoded)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}

function getDuration(startedAt: string, completedAt: string): string {
  const ms = new Date(completedAt).getTime() - new Date(startedAt).getTime()
  const hours = Math.floor(ms / 3600000)
  const minutes = Math.floor((ms % 3600000) / 60000)
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}
