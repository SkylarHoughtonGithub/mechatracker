'use client'

import Link from 'next/link'
import { useGarage } from '@/lib/queries'
import { UserButton } from '@clerk/nextjs'

export default function HomePage() {
  const { data, isLoading, error } = useGarage()

  if (isLoading) return <div className="p-8 text-sm text-gray-500">Loading garage...</div>
  if (error) return <div className="p-8 text-sm text-red-500">Error: {error.message}</div>

  return (
    <main className="p-8 max-w-2xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Garage</h1>
        <UserButton />
      </div>

      <div className="grid grid-cols-4 gap-3 mb-8">
        <Stat label="Vehicles"       value={data?.stats?.total_vehicles ?? 0} />
        <Stat label="Open repairs"   value={data?.stats?.open_repairs ?? 0} />
        <Stat label="Repairs (30d)"  value={data?.stats?.repairs_30d ?? 0} />
        <Stat label="Parts spend"    value={`$${data?.stats?.parts_spend_30d ?? 0}`} />
      </div>

      <div className="flex flex-col gap-2">
        {data?.vehicles?.map(vehicle => (
          <Link
            key={vehicle.id}
            href={`/vehicles/${vehicle.id}`}
            className="flex items-center justify-between rounded-lg border p-4 hover:bg-gray-50 transition-colors"
          >
            <div>
              <div className="font-medium text-sm">
                {vehicle.year} {vehicle.make} {vehicle.model}
                {vehicle.trim && <span className="text-gray-400"> · {vehicle.trim}</span>}
              </div>
              {vehicle.mileage && (
                <div className="text-xs text-gray-400 mt-0.5">
                  {vehicle.mileage.toLocaleString()} mi
                </div>
              )}
            </div>
            <div className="flex items-center gap-3 text-xs">
              {(vehicle.open_repairs ?? 0) > 0 && (
                <span className="bg-blue-50 text-blue-700 px-2 py-0.5 rounded-full font-medium">
                  {vehicle.open_repairs} open
                </span>
              )}
              {(vehicle.systems?.length ?? 0) > 0 && (
                <span className="text-gray-400">
                  {vehicle.systems?.length} system{vehicle.systems?.length !== 1 ? 's' : ''}
                </span>
              )}
              <svg className="w-4 h-4 text-gray-300" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
              </svg>
            </div>
          </Link>
        ))}
      </div>
    </main>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border p-3">
      <div className="text-xs text-gray-400 mb-1">{label}</div>
      <div className="text-xl font-semibold">{value}</div>
    </div>
  )
}