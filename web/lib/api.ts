import type { components } from '../../shared/types'

// Convenience aliases
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

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(`${API_URL}${path}`,{
        ...init,
        headers: {
            'Content-Type': 'application/json',
            ...init?.headers,
        },
    })

    if (!res.ok) {
        const error = await res.json().catch(() => ({}))
        throw new Error(error?.error?.message ?? `API error ${res.status}`)
    }

    return res.json()
}

export const api = {
    garage: {
        get: () => apiFetch<GarageResponse>('/v1/garage'),
    },
    vehicles: {
        list: () => apiFetch<GarageVehicle[]>('/v1/vehicles'),
    },
}