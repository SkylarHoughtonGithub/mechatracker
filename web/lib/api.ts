import { useAuth } from '@clerk/nextjs'
import type { components } from '../../shared/types'

export type GarageResponse = components['schemas']['internal_handler.garageResponse']
export type GarageVehicle  = components['schemas']['internal_handler.garageVehicle']
export type GarageStats    = components['schemas']['internal_handler.garageStats']
export type Vehicle        = components['schemas']['internal_handler.SwaggerVehicle']
export type Repair         = components['schemas']['internal_handler.SwaggerRepair']
export type RepairDetail   = components['schemas']['internal_handler.SwaggerRepairDetail']
export type RepairStep     = components['schemas']['internal_handler.SwaggerRepairStep']
export type Part           = components['schemas']['internal_handler.SwaggerPart']
export type RepairNote     = components['schemas']['internal_handler.SwaggerRepairNote']
export type SystemSummary  = components['schemas']['internal_handler.garageSystemSummary']

export type NoteWithAuthor = RepairNote & { author_name?: string }

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

async function apiFetch<T>(path: string, token: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
      ...init?.headers,
    },
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({}))
    throw new Error(error?.error?.message ?? `API error ${res.status}`)
  }

  return res.json()
}

export function useApi() {
  const { getToken } = useAuth()

  async function fetchApi<T>(path: string, init?: RequestInit): Promise<T> {
    const token = await getToken()
    if (!token) throw new Error('Not authenticated')
    return apiFetch<T>(path, token, init)
  }

  return {
    garage: {
      get: () => fetchApi<GarageResponse>('/v1/garage'),
    },

    vehicles: {
      list: () => fetchApi<Vehicle[]>('/v1/vehicles'),
      get:  (id: string) => fetchApi<Vehicle>(`/v1/vehicles/${id}`),
      repairs: (id: string, system?: string) => {
        const qs = system ? `?system=${encodeURIComponent(system)}` : ''
        return fetchApi<Repair[]>(`/v1/vehicles/${id}/repairs${qs}`)
      },
    },

    repairs: {
      get: (repairId: string) =>
        fetchApi<RepairDetail>(`/v1/repairs/${repairId}`),

      getNotes: (repairId: string) =>
        fetchApi<NoteWithAuthor[]>(`/v1/repairs/${repairId}/notes`),

      updateStatus: (repairId: string, status: string) =>
        fetchApi<Repair>(`/v1/repairs/${repairId}`, {
          method: 'PATCH',
          body: JSON.stringify({ status }),
        }),

      toggleStep: (repairId: string, stepId: string) =>
        fetchApi<{ step: RepairStep; repair: Repair }>(
          `/v1/repairs/${repairId}/steps/${stepId}/complete`,
          { method: 'POST' }
        ),

      updatePartStatus: (repairId: string, partId: string, status: string) =>
        fetchApi<Part>(`/v1/repairs/${repairId}/parts/${partId}/status`, {
          method: 'PATCH',
          body: JSON.stringify({ status }),
        }),

      createNote: (repairId: string, body: string) =>
        fetchApi<RepairNote>(`/v1/repairs/${repairId}/notes`, {
          method: 'POST',
          body: JSON.stringify({ body }),
        }),
    },
  }
}