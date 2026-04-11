'use client'

import { useGarage } from '@/lib/queries'

export default function HomePage() {
  const { data, isLoading, error } = useGarage()

  if (isLoading) return <div className="p-8">Loading garage...</div>
  if (error) return <div className="p-8 text-red-500">Error: {error.message}</div>

  return (
    <main className="p-8">
      <h1 className="text-2xl font-semibold mb-6">Garage</h1>

      <div className="grid grid-cols-4 gap-4 mb-8">
        <div className="rounded-lg border p-4">
          <div className="text-sm text-gray-500">Vehicles</div>
          <div className="text-2xl font-semibold">{data?.stats?.total_vehicles}</div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-sm text-gray-500">Open Repairs</div>
          <div className="text-2xl font-semibold">{data?.stats?.open_repairs}</div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-sm text-gray-500">Repairs (30d)</div>
          <div className="text-2xl font-semibold">{data?.stats?.repairs_30d}</div>
        </div>
        <div className="rounded-lg border p-4">
          <div className="text-sm text-gray-500">Parts Spend (30d)</div>
          <div className="text-2xl font-semibold">${data?.stats?.parts_spend_30d}</div>
        </div>
      </div>

      <div className="flex flex-col gap-3">
        {data?.vehicles?.map(vehicle => (
          <div key={vehicle.id} className="rounded-lg border p-4">
            <div className="font-medium">
              {vehicle.year} {vehicle.make} {vehicle.model}
              {vehicle.trim && <span className="text-gray-500"> {vehicle.trim}</span>}
            </div>
            <div className="text-sm text-gray-500 mt-1">
              {(vehicle.open_repairs ?? 0) > 0
                ? `${vehicle.open_repairs} open repair${(vehicle.open_repairs ?? 0) > 1 ? 's' : ''}`
                : 'No open repairs'}
            </div>
          </div>
        ))}
      </div>
    </main>
  )
}