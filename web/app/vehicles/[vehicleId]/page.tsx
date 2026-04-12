'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useGarage } from '@/lib/queries'

const SYSTEM_ICONS: Record<string, string> = {
  engine:       '⚙️',
  suspension:   '🔩',
  brakes:       '🛑',
  electrical:   '⚡',
  transmission: '🔧',
  cooling:      '🌡️',
  fuel:         '⛽',
  exhaust:      '💨',
  steering:     '🎯',
  tires:        '🔄',
  hvac:         '❄️',
  body:         '🚗',
}

export default function VehiclePage() {
  const params = useParams()
  const vehicleId = params.vehicleId as string

  const { data: garage, isLoading, error } = useGarage()

  if (isLoading) return <div className="p-8 text-sm text-gray-500">Loading...</div>
  if (error) return <div className="p-8 text-sm text-red-500">Error: {(error as Error).message}</div>

  const vehicle = garage?.vehicles?.find(v => v.id === vehicleId)
  if (!vehicle) return <div className="p-8 text-sm text-gray-400">Vehicle not found.</div>

  const label = `${vehicle.year} ${vehicle.make} ${vehicle.model}${vehicle.trim ? ` · ${vehicle.trim}` : ''}`

  return (
    <main className="p-8 max-w-2xl">
      <nav className="flex items-center gap-1.5 text-xs text-gray-400 mb-6">
        <Link href="/" className="hover:text-gray-700">Garage</Link>
        <span>›</span>
        <span className="text-gray-700">{label}</span>
      </nav>

      <div className="rounded-lg border p-4 mb-6">
        <div className="font-semibold text-lg">{label}</div>
        {vehicle.mileage != null && (
          <div className="text-sm text-gray-400 mt-0.5">{vehicle.mileage.toLocaleString()} mi</div>
        )}
        {(vehicle.open_repairs ?? 0) > 0 && (
          <div className="mt-3">
            <span className="bg-blue-50 text-blue-700 text-xs px-2 py-0.5 rounded-full font-medium">
              {vehicle.open_repairs} open repair{vehicle.open_repairs !== 1 ? 's' : ''}
            </span>
          </div>
        )}
      </div>

      <h2 className="text-sm font-medium text-gray-500 mb-3">Systems</h2>

      {(!vehicle.systems || vehicle.systems.length === 0) ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-gray-400">
          No repairs logged yet for this vehicle.
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {vehicle.systems.map(system => (
            <Link
              key={system.id}
              href={`/vehicles/${vehicleId}/${system.id}`}
              className="flex items-center gap-3 rounded-lg border p-4 hover:border-gray-300 hover:bg-gray-50 transition-colors"
            >
              <div className="text-2xl w-8 text-center flex-shrink-0">
                {SYSTEM_ICONS[system.id ?? ''] ?? '🔧'}
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-medium text-sm">{system.label}</div>
                <div className="text-xs text-gray-400 mt-0.5">
                  {system.repair_count} repair{system.repair_count !== 1 ? 's' : ''}
                </div>
              </div>
              <div className="flex items-center gap-2 text-xs flex-shrink-0">
                {system.has_open && (
                  <span className="bg-blue-50 text-blue-700 px-2 py-0.5 rounded-full font-medium">
                    open
                  </span>
                )}
                <svg className="w-4 h-4 text-gray-300" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                </svg>
              </div>
            </Link>
          ))}
        </div>
      )}
    </main>
  )
}