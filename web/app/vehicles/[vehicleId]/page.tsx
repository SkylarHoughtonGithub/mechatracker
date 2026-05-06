'use client'

import { useState, useEffect, useRef } from 'react'
import EmojiPicker, { type EmojiClickData } from 'emoji-picker-react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useQueryClient } from '@tanstack/react-query'
import { useGarage } from '@/lib/queries'
import { useApi, type GarageVehicle } from '@/lib/api'
import { DarkModal, CancelButton, ConfirmModal } from '@/app/components/modal'
import { Chevron } from '@/app/components/chevron'
import { PageSpinner } from '@/app/components/spinner'

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

type VehiclePrefs = {
  hiddenSystems: string[]
  systemLabels: Record<string, string>
  systemIcons: Record<string, string>
}

function prefsKey(vehicleId: string) { return `mt:vprefs:${vehicleId}` }

function loadPrefs(vehicleId: string): VehiclePrefs {
  try {
    const raw = localStorage.getItem(prefsKey(vehicleId))
    if (!raw) return { hiddenSystems: [], systemLabels: {}, systemIcons: {} }
    const p = JSON.parse(raw)
    return { hiddenSystems: p.hiddenSystems ?? [], systemLabels: p.systemLabels ?? {}, systemIcons: p.systemIcons ?? {} }
  } catch { return { hiddenSystems: [], systemLabels: {}, systemIcons: {} } }
}

function savePrefs(vehicleId: string, prefs: VehiclePrefs) {
  localStorage.setItem(prefsKey(vehicleId), JSON.stringify(prefs))
}

export default function VehiclePage() {
  const params = useParams()
  const vehicleId = params.vehicleId as string
  const router = useRouter()
  const api = useApi()
  const qc = useQueryClient()

  const { data: garage, isLoading, error } = useGarage()

  const [customModalOpen, setCustomModalOpen]   = useState(false)
  const [customSystem, setCustomSystem]         = useState('')
  const [editVehicleOpen, setEditVehicleOpen]   = useState(false)
  const [deleteVehicleOpen, setDeleteVehicleOpen] = useState(false)
  const [manageMode, setManageMode]             = useState(false)
  const [editingSystem, setEditingSystem]       = useState<{ id: string; label: string; icon: string } | null>(null)
  const [deleteSystemId, setDeleteSystemId]     = useState<string | null>(null)
  const [prefs, setPrefs]                       = useState<VehiclePrefs>({ hiddenSystems: [], systemLabels: {}, systemIcons: {} })

  useEffect(() => { setPrefs(loadPrefs(vehicleId)) }, [vehicleId])

  useEffect(() => {
    const v = garage?.vehicles?.find(v => v.id === vehicleId)
    if (v && garage?.org_name) {
      const lbl = `${v.year} ${v.make} ${v.model}${v.trim ? ` · ${v.trim}` : ''}`
      document.title = `${lbl} · ${garage.org_name}`
    }
  }, [garage, vehicleId])

  if (isLoading) return <PageSpinner />
  if (error)     return (
    <div className="p-8 text-sm text-red-500 text-center">
      <p>Failed to load vehicle.</p>
      <p className="text-zinc-600 mt-1">{(error as Error).message}</p>
    </div>
  )

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
    .map(s => ({ id: s.id!, label: s.label ?? s.id!, icon: '🔨', isCustom: true }))

  const allDisplayed = [
    ...ALL_SYSTEMS.map(s => ({ ...s, isCustom: false })),
    ...customSystems,
  ]

  const visibleSystems = manageMode
    ? allDisplayed
    : allDisplayed.filter(s => !prefs.hiddenSystems.includes(s.id))

  function updatePrefs(next: VehiclePrefs) {
    setPrefs(next)
    savePrefs(vehicleId, next)
  }

  function toggleSystemVisibility(systemId: string) {
    const hidden = prefs.hiddenSystems.includes(systemId)
      ? prefs.hiddenSystems.filter(id => id !== systemId)
      : [...prefs.hiddenSystems, systemId]
    updatePrefs({ ...prefs, hiddenSystems: hidden })
  }

  function saveSystemSettings(systemId: string, newLabel: string, newIcon: string) {
    const systemLabels = { ...prefs.systemLabels, [systemId]: newLabel.trim() || systemId }
    const systemIcons  = { ...prefs.systemIcons,  [systemId]: newIcon.trim()  || '🔨'    }
    updatePrefs({ ...prefs, systemLabels, systemIcons })
    setEditingSystem(null)
  }

  async function handleDeleteVehicle() {
    await api.vehicles.delete(vehicleId)
    localStorage.removeItem(prefsKey(vehicleId))
    await qc.invalidateQueries({ queryKey: ['garage'] })
    router.push('/')
  }

  async function handleDeleteSystem(systemId: string) {
    await api.vehicles.deleteSystem(vehicleId, systemId)
    const systemLabels = { ...prefs.systemLabels }
    const systemIcons  = { ...prefs.systemIcons  }
    delete systemLabels[systemId]
    delete systemIcons[systemId]
    updatePrefs({ ...prefs, systemLabels, systemIcons })
    setDeleteSystemId(null)
    await qc.invalidateQueries({ queryKey: ['garage'] })
  }

  function goToCustomSystem() {
    const id = customSystem.trim().toLowerCase().replace(/\s+/g, '_')
    if (!id) return
    setCustomModalOpen(false)
    setCustomSystem('')
    router.push(`/vehicles/${vehicleId}/${id}`)
  }

  return (
    <div className="px-4 py-6 sm:p-8">
      <div className="max-w-2xl mx-auto">
        <nav className="flex items-center gap-1.5 text-xs text-zinc-500 mb-6">
          <Link href="/" className="hover:text-zinc-300 transition-colors">{garage?.org_name ?? 'Garage'}</Link>
          <Chevron />
          <span className="text-zinc-200">{label}</span>
        </nav>

        {/* Vehicle card */}
        <div className="bg-zinc-900 rounded-xl border border-zinc-800 p-4 mb-6">
          <div className="flex items-start justify-between">
            <div>
              <div className="font-semibold text-lg text-zinc-100">{label}</div>
              {vehicle.mileage != null && (
                <div className="text-sm text-zinc-500 mt-0.5">{vehicle.mileage.toLocaleString()} mi</div>
              )}
            </div>
            <div className="flex items-center gap-0.5">
              <button
                onClick={() => setEditVehicleOpen(true)}
                title="Edit vehicle"
                aria-label="Edit vehicle"
                className="p-1.5 text-zinc-600 hover:text-zinc-300 transition-colors rounded-lg hover:bg-zinc-800"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125" />
                </svg>
              </button>
              <button
                onClick={() => setDeleteVehicleOpen(true)}
                title="Delete vehicle"
                aria-label="Delete vehicle"
                className="p-1.5 text-zinc-600 hover:text-red-400 transition-colors rounded-lg hover:bg-zinc-800"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                </svg>
              </button>
            </div>
          </div>
          {(vehicle.open_repairs ?? 0) > 0 && (
            <div className="mt-3">
              <span className="bg-blue-950 text-blue-400 text-xs px-2 py-0.5 rounded-full font-medium">
                {vehicle.open_repairs} open repair{vehicle.open_repairs !== 1 ? 's' : ''}
              </span>
            </div>
          )}
        </div>

        {/* Systems header */}
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-xs font-medium text-zinc-500 uppercase tracking-wide">Systems</h2>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setManageMode(m => !m)}
              className={`text-xs font-medium transition-colors ${manageMode ? 'text-blue-400' : 'text-zinc-500 hover:text-zinc-300'}`}
            >
              {manageMode ? 'Done' : 'Manage'}
            </button>
            <Link href={`/vehicles/${vehicleId}/history`} className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors">
              View history →
            </Link>
            <button onClick={() => setCustomModalOpen(true)} className="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors">
              + Add system
            </button>
          </div>
        </div>

        {/* Systems list */}
        <div className="flex flex-col gap-1.5">
          {visibleSystems.map(system => {
            const stats = systemStats[system.id]
            const hasRepairs = (stats?.repair_count ?? 0) > 0
            const isHidden = prefs.hiddenSystems.includes(system.id)
            const displayLabel = prefs.systemLabels[system.id] ?? system.label
            const displayIcon  = prefs.systemIcons[system.id]  ?? system.icon

            if (manageMode) {
              return (
                <div
                  key={system.id}
                  className={`flex items-center gap-3 bg-zinc-900 rounded-xl border border-zinc-800 p-4 transition-opacity ${isHidden ? 'opacity-40' : ''}`}
                >
                  <button
                    onClick={() => toggleSystemVisibility(system.id)}
                    title={isHidden ? 'Show system' : 'Hide system'}
                    aria-label={isHidden ? `Show ${displayLabel}` : `Hide ${displayLabel}`}
                    className="text-zinc-500 hover:text-zinc-200 transition-colors flex-shrink-0"
                  >
                    {isHidden ? (
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 10-4.243-4.243m4.242 4.242L9.88 9.88" />
                      </svg>
                    ) : (
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
                        <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      </svg>
                    )}
                  </button>

                  <div className="text-xl w-7 text-center flex-shrink-0">{displayIcon}</div>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-sm text-zinc-100">{displayLabel}</div>
                    <div className="text-xs mt-0.5">
                      {hasRepairs
                        ? <span className="text-zinc-500">{stats.repair_count} repair{stats.repair_count !== 1 ? 's' : ''}</span>
                        : <span className="text-zinc-700">No repairs yet</span>
                      }
                    </div>
                  </div>

                  <div className="flex items-center gap-1 flex-shrink-0">
                    {stats?.has_open && (
                      <span className="text-xs bg-blue-950 text-blue-400 px-2 py-0.5 rounded-full font-medium mr-1">open</span>
                    )}
                    <button
                      onClick={() => setEditingSystem({ id: system.id, label: displayLabel, icon: displayIcon })}
                      title="Edit system"
                      aria-label={`Edit ${displayLabel}`}
                      className="p-1.5 text-zinc-600 hover:text-zinc-300 transition-colors rounded-lg hover:bg-zinc-800"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />
                      </svg>
                    </button>
                    {system.isCustom && (
                      <button
                        onClick={() => setDeleteSystemId(system.id)}
                        title="Delete system"
                        aria-label={`Delete ${displayLabel}`}
                        className="p-1.5 text-zinc-600 hover:text-red-400 transition-colors rounded-lg hover:bg-zinc-800"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                        </svg>
                      </button>
                    )}
                  </div>
                </div>
              )
            }

            return (
              <Link
                key={system.id}
                href={`/vehicles/${vehicleId}/${system.id}`}
                className="flex items-center gap-3 bg-zinc-900 rounded-xl border border-zinc-800 p-4 hover:border-zinc-700 hover:bg-zinc-800/50 transition-colors"
              >
                <div className="text-xl w-7 text-center flex-shrink-0">{displayIcon}</div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-sm text-zinc-100">{displayLabel}</div>
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
        <DarkModal title="Add system" onClose={() => setCustomModalOpen(false)}>
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
          <p className="text-xs text-zinc-600 mt-2 mb-5">You&apos;ll land on the repair list for this system.</p>
          <div className="flex gap-2">
            <CancelButton onClick={() => setCustomModalOpen(false)} />
            <button onClick={goToCustomSystem} disabled={!customSystem.trim()} className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors">
              Go to system
            </button>
          </div>
        </DarkModal>
      )}

      {editVehicleOpen && (
        <EditVehicleModal vehicle={vehicle} onClose={() => setEditVehicleOpen(false)} />
      )}

      {deleteVehicleOpen && (
        <ConfirmModal
          title="Delete vehicle"
          message={`Delete ${label}? This will permanently remove the vehicle and all its repairs.`}
          confirmLabel="Delete vehicle"
          danger
          onConfirm={handleDeleteVehicle}
          onClose={() => setDeleteVehicleOpen(false)}
        />
      )}

      {editingSystem && (
        <EditSystemModal
          system={editingSystem}
          onSave={(newLabel, newIcon) => saveSystemSettings(editingSystem.id, newLabel, newIcon)}
          onClose={() => setEditingSystem(null)}
        />
      )}

      {deleteSystemId && (() => {
        const sys = allDisplayed.find(s => s.id === deleteSystemId)
        const displayLabel = prefs.systemLabels[deleteSystemId] ?? sys?.label ?? deleteSystemId
        const repairCount = systemStats[deleteSystemId]?.repair_count ?? 0
        return (
          <ConfirmModal
            title="Delete system"
            message={`Delete "${displayLabel}"? This will permanently remove all ${repairCount} repair${repairCount !== 1 ? 's' : ''} in this system.`}
            confirmLabel="Delete system"
            danger
            onConfirm={() => handleDeleteSystem(deleteSystemId)}
            onClose={() => setDeleteSystemId(null)}
          />
        )
      })()}
    </div>
  )
}

function EditVehicleModal({ vehicle, onClose }: { vehicle: GarageVehicle; onClose: () => void }) {
  const api = useApi()
  const qc = useQueryClient()
  const [year, setYear]     = useState(String(vehicle.year ?? ''))
  const [make, setMake]     = useState(vehicle.make ?? '')
  const [model, setModel]   = useState(vehicle.model ?? '')
  const [trim, setTrim]     = useState(vehicle.trim ?? '')
  const [mileage, setMileage] = useState(vehicle.mileage != null ? String(vehicle.mileage) : '')
  const [saving, setSaving] = useState(false)
  const [err, setErr]       = useState('')

  async function submit() {
    const yearNum = parseInt(year, 10)
    if (!make.trim() || !model.trim() || isNaN(yearNum)) { setErr('Year, make, and model are required.'); return }
    setSaving(true); setErr('')
    try {
      await api.vehicles.update(vehicle.id!, {
        year: yearNum,
        make: make.trim(),
        model: model.trim(),
        trim: trim.trim() || undefined,
        mileage: mileage ? parseInt(mileage, 10) : undefined,
      })
      await qc.invalidateQueries({ queryKey: ['garage'] })
      onClose()
    } catch (e) { setErr(e instanceof Error ? e.message : 'Something went wrong.') }
    finally { setSaving(false) }
  }

  return (
    <DarkModal title="Edit vehicle" onClose={onClose}>
      <div className="space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
          {[
            { label: 'Year *',  value: year,  set: setYear,  placeholder: '2019', type: 'number' },
            { label: 'Make *',  value: make,  set: setMake,  placeholder: 'Ford',  type: 'text'   },
            { label: 'Model *', value: model, set: setModel, placeholder: 'F-150', type: 'text'   },
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
            { label: 'Trim',    value: trim,    set: setTrim,    placeholder: 'XLT'   },
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
            {saving ? 'Saving…' : 'Save changes'}
          </button>
        </div>
      </div>
    </DarkModal>
  )
}

function EditSystemModal({ system, onSave, onClose }: {
  system: { id: string; label: string; icon: string }
  onSave: (label: string, icon: string) => void
  onClose: () => void
}) {
  const [label, setLabel]       = useState(system.label)
  const [icon, setIcon]         = useState(system.icon)
  const [pickerOpen, setPickerOpen] = useState(false)
  const pickerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setPickerOpen(false)
      }
    }
    if (pickerOpen) document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [pickerOpen])

  function submit() {
    if (!label.trim()) return
    onSave(label, icon)
  }

  return (
    <DarkModal title="Edit system" onClose={onClose}>
      <div className="flex gap-3 mb-3">
        <div className="flex-shrink-0">
          <label className="block text-xs text-zinc-500 mb-1.5">Icon</label>
          <div className="relative" ref={pickerRef}>
            <button
              onClick={() => setPickerOpen(o => !o)}
              aria-label="Choose emoji"
              className="w-12 h-10 bg-zinc-800 border border-zinc-700 rounded-xl text-xl flex items-center justify-center hover:border-zinc-500 transition-colors"
            >
              {icon}
            </button>
            {pickerOpen && (
              <div className="absolute top-12 left-0 z-50">
                <EmojiPicker
                  onEmojiClick={(e: EmojiClickData) => { setIcon(e.emoji); setPickerOpen(false) }}
                  theme={'dark' as Parameters<typeof EmojiPicker>[0]['theme']}
                  skinTonesDisabled
                  searchPlaceholder="Search emoji…"
                  height={380}
                  width={320}
                />
              </div>
            )}
          </div>
        </div>
        <div className="flex-1">
          <label className="block text-xs text-zinc-500 mb-1.5">Display name</label>
          <input
            type="text"
            value={label}
            onChange={e => setLabel(e.target.value)}
            autoFocus
            onKeyDown={e => { if (e.key === 'Enter') submit() }}
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>
      <div className="flex gap-2 mt-5">
        <CancelButton onClick={onClose} />
        <button onClick={submit} disabled={!label.trim()} className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors">
          Save
        </button>
      </div>
    </DarkModal>
  )
}
