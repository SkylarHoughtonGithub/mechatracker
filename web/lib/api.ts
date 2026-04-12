import { useAuth } from '@clerk/nextjs';
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

    async function fetch<T>(path: string, init?: RequestInit): Promise<T> {
        const token = await getToken()
        if (!token) throw new Error('Not authenticated')
        return apiFetch<T>(path, token, init)
    }

    return {
        garage: {
            get: () => fetch<GarageResponse>('/v1/garage'),
        },
        vehicles: {
            list: () => fetch<Vehicle[]>('/v1/vehicles'),
            get: (id: string) => fetch<Vehicle>(`/v1/vehicles/${id}`),
            repairs: (id: string, system?: string) => {
                const qs = system ? `?system=${encodeURIComponent(system)}` : ''
                return fetch<Repair[]>(`/v1/vehicles/${id}/repairs${qs}`)
            },
        },
    }
}