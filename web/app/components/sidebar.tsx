'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useGarage } from '@/lib/queries'

export function Sidebar() {
  const params = useParams()
  const activeVehicleId = params?.vehicleId as string | undefined
  const activeSystem = params?.system as string | undefined

  const { data: garage } = useGarage()
  const vehicles = garage?.vehicles ?? []

  return (
    <nav
      className="w-52 shrink-0 border-r border-zinc-800 bg-zinc-950 sticky top-12 self-start h-[calc(100vh-3rem)] overflow-y-auto hidden md:block"
      aria-label="Vehicle navigation"
    >
      <div className="p-3 pt-5">
        <p className="text-[10px] font-semibold text-zinc-600 uppercase tracking-wider px-2 mb-2">
          Vehicles
        </p>

        {vehicles.length === 0 && (
          <p className="text-xs text-zinc-700 px-2 py-1">No vehicles yet</p>
        )}

        <div className="space-y-0.5">
          {vehicles.map(vehicle => {
            const isActiveVehicle = vehicle.id === activeVehicleId
            const systems = (vehicle.systems ?? []).filter(s => s.id)

            return (
              <div key={vehicle.id}>
                <Link
                  href={`/vehicles/${vehicle.id}`}
                  className={`flex items-start justify-between w-full px-2 py-2 rounded-lg text-xs transition-colors
                    ${isActiveVehicle
                      ? 'bg-zinc-800 text-zinc-100'
                      : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}`}
                >
                  <div className="min-w-0">
                    <div className="font-medium truncate leading-tight">
                      {vehicle.year} {vehicle.make} {vehicle.model}
                    </div>
                    {vehicle.trim && (
                      <div className="text-zinc-600 truncate mt-0.5">{vehicle.trim}</div>
                    )}
                  </div>
                  {(vehicle.open_repairs ?? 0) > 0 && (
                    <span className="ml-2 shrink-0 text-blue-400 font-medium tabular-nums">
                      {vehicle.open_repairs}
                    </span>
                  )}
                </Link>

                {isActiveVehicle && systems.length > 0 && (
                  <div className="ml-3 mt-0.5 mb-1 pl-2.5 border-l border-zinc-800 space-y-0.5">
                    {systems.map(sys => {
                      const isActiveSystem = sys.id === activeSystem
                      return (
                        <Link
                          key={sys.id}
                          href={`/vehicles/${vehicle.id}/${sys.id}`}
                          className={`flex items-center justify-between px-2 py-1.5 rounded-md text-xs transition-colors
                            ${isActiveSystem
                              ? 'bg-zinc-800 text-zinc-100'
                              : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'}`}
                        >
                          <span className="truncate">{sys.label ?? sys.id}</span>
                          {sys.has_open && (
                            <span
                              className="ml-2 w-1.5 h-1.5 rounded-full bg-blue-400 shrink-0"
                              aria-label="Has open repairs"
                            />
                          )}
                        </Link>
                      )
                    })}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </nav>
  )
}
