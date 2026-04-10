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

// ── Types ─────────────────────────────────────────────────────────────────────

export interface SystemSummary {
    id: string
    label: string
    repair_count: number
    has_open: boolean
}

export interface GarageVehicle {
    id: string
    year: number
    make: string
    model: string
    trim?: string
    open_repairs: number
    systems: SystemSummary[]
}

export interface GarageStats {
    total_vehicles: number
    open_repairs: number
    repairs_30d: number
    parts_spend_30d: number
}

export interface GarageResponse {
    vehicles: GarageVehicle[]
    stats: GarageStats
}

// ── Endpoints ─────────────────────────────────────────────────────────────────

export const api = {
    garage: {
        get: () => apiFetch<GarageResponse>('/v1/garage'),
    },
    vehicles: {
        list: () => apiFetch<GarageVehicle[]>('/v1/vehicles'),
    },
}