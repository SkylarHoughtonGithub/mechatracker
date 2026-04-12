'use client'

import { use } from 'react'
import Link from 'next/link'
import { useGarage, useVehicleRepairs } from '@/lib/queries'
import type { Repair } from '@/lib/api'

const STATUS_LABEL: Record<string, string> = {
  draft:          'Draft',
  in_progress:    'In progress',
  pending_parts:  'Pending parts',
  completed:      'Completed',
}

const STATUS_STYLE: Record<string, string> = {
  draft:          'bg-gray-100 text-gray-500',
  in_progress:    'bg-blue-50 text-blue-700',
  pending_parts:  'bg-amber-50 text-amber-700',
  completed:      'bg-green-50 text-green-700',
}

export default function SystemRepairsPage({
  params,
}: {
  params: Promise<{ vehicleId: string; system: string }>
}) {
  const { vehicleId, system } = use(params)
  const { data: garage, isLoading: garageLoading } = useGarage()
  const { data: repairs, isLoading: repairsLoading, error } = useVehicleRepairs(vehicleId, system)

  const isLoading = garageLoading || repairsLoading

  if (isLoading) return <div className="p-8 text-sm text-gray-500">Loading...</div>
  if (error) return <div className="p-8 text-sm text-red-500">Error: {error.message}</div>

  const vehicle = garage?.vehicles?.find(v => v.id === vehicleId)
  const systemInfo = vehicle?.systems?.find(s => s.id === system)

  const vehicleLabel = vehicle
    ? `${vehicle.year} ${vehicle.make} ${vehicle.model}${vehicle.trim ? ` ${vehicle.trim}` : ''}`
    : '...'
  const systemLabel = systemInfo?.label ?? system

  return (
    <main className="p-8 max-w-2xl">
      <nav className="flex items-center gap-1.5 text-xs text-gray-400 mb-6 flex-wrap">
        <Link href="/" className="hover:text-gray-700">Garage</Link>
        <span></span>
        <Link href={`/vehicles/${vehicleId}`} className="hover:text-gray-700">{vehicleLabel}</Link>
        <span></span>
        <span className="text-gray-700">{systemLabel}</span>
      </nav>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold">{systemLabel}</h1>
          <div className="text-sm text-gray-400 mt-0.5">{vehicleLabel}</div>
        </div>
      </div>

      {repairs?.length === 0 ? (
        <div className="text-sm text-gray-400 py-8 text-center border rounded-lg">
          No repairs logged for this system yet.
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {repairs?.map(repair => (
            <RepairRow key={repair.id} repair={repair} vehicleId={vehicleId} system={system} />
          ))}
        </div>
      )}
    </main>
  )
}

function RepairRow({
  repair,
  vehicleId,
  system,
}: {
  repair: Repair
  vehicleId: string
  system: string
}) {
  const status = repair.status ?? 'draft'
  const progress = repair.progress_pct ?? 0

  return (
    <Link
      href={`/vehicles/${vehicleId}/${system}/${repair.id}`}
      className="rounded-lg border p-4 hover:bg-gray-50 transition-colors block"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm">{repair.title}</div>
          {repair.started_at && (
            <div className="text-xs text-gray-400 mt-0.5">
              Started {new Date(repair.started_at).toLocaleDateString()}
            </div>
          )}
          {repair.completed_at && (
            <div className="text-xs text-gray-400 mt-0.5">
              Completed {new Date(repair.completed_at).toLocaleDateString()}
            </div>
          )}
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_STYLE[status] ?? STATUS_STYLE.draft}`}>
            {STATUS_LABEL[status] ?? status}
          </span>
          <svg className="w-4 h-4 text-gray-300" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
          </svg>
        </div>
      </div>

      {status === 'in_progress' && progress > 0 && (
        <div className="mt-3">
          <div className="flex justify-between text-xs text-gray-400 mb-1">
            <span>Progress</span>
            <span>{progress}%</span>
          </div>
          <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      )}

      {(repair.total_parts_cost ?? 0) > 0 && (
        <div className="mt-2 text-xs text-gray-400">
          Parts: <span className="text-gray-600 font-medium">${repair.total_parts_cost}</span>
        </div>
      )}
    </Link>
  )
}