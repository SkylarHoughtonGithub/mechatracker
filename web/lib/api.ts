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
export type RepairHistory  = components['schemas']['internal_handler.SwaggerRepairHistory']

export type NoteWithAuthor = RepairNote & { author_name?: string }

// Everything goes through Next.js's rewrite (/api/backend/* → BACKEND_URL)
// so requests stay same-origin and the httpOnly session cookie flows
// without SameSite=None / Secure gymnastics in dev.
const API_BASE = '/api/backend'

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (init?.headers) {
    Object.assign(headers, init.headers as Record<string, string>)
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    credentials: 'include',
  })

  if (!res.ok) {
    let message = `API error ${res.status}`
    try {
      const body = await res.json()
      message = body?.error?.message ?? body?.message ?? message
    } catch {
      // non-JSON body
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

// ── Auth / onboarding ────────────────────────────────────────────────────────

export type MeUser = {
  id: string
  email: string
  email_verified: boolean
  name: string
  picture_url?: string
  role: 'owner' | 'admin' | 'member'
  org_id: string
}

export type MeOrg = {
  id: string
  name: string
  slug: string
  plan: string
}

export type MeGoogleClaims = {
  sub: string
  email: string
  email_verified: boolean
  name: string
  picture_url?: string
}

export type MeResponse = {
  user: MeUser | null
  org?: MeOrg | null
  needs_onboarding?: boolean
  google?: MeGoogleClaims
}

export const auth = {
  // Google ID token → session cookie. Server returns the full /auth/me
  // shape so the caller can route to / or /onboarding without a second RTT.
  google: (idToken: string) =>
    apiFetch<MeResponse>('/auth/google', {
      method: 'POST',
      headers: { Authorization: `Bearer ${idToken}` },
    }),
  me: () => apiFetch<MeResponse>('/auth/me'),
  logout: () => apiFetch<void>('/auth/logout', { method: 'POST' }),
}

export const onboarding = {
  createOrg: (name: string) =>
    apiFetch<MeResponse>('/orgs', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  acceptInvite: (token: string) =>
    apiFetch<MeResponse>(`/invites/${encodeURIComponent(token)}/accept`, {
      method: 'POST',
    }),
}

// ── Tenant API ───────────────────────────────────────────────────────────────
//
// Plain module-level functions because auth travels in the mt_session cookie,
// which the browser attaches automatically. Existing callers that invoke
// `useApi()` still work via the shim at the bottom of this file.

export const api = {
  garage: {
    get: () => apiFetch<GarageResponse>('/v1/garage'),
  },

  vehicles: {
    list: () => apiFetch<Vehicle[]>('/v1/vehicles'),
    get: (id: string) => apiFetch<Vehicle>(`/v1/vehicles/${id}`),
    repairs: (id: string, system?: string) => {
      const qs = system ? `?system=${encodeURIComponent(system)}` : ''
      return apiFetch<Repair[]>(`/v1/vehicles/${id}/repairs${qs}`)
    },
    history: (id: string) =>
      apiFetch<RepairHistory[]>(`/v1/vehicles/${id}/history`),
    create: (body: {
      year: number; make: string; model: string; trim?: string; mileage?: number
    }) =>
      apiFetch<Vehicle>('/v1/vehicles', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    update: (id: string, body: {
      year?: number; make?: string; model?: string; trim?: string; mileage?: number; vin?: string; unit_label?: string
    }) =>
      apiFetch<Vehicle>(`/v1/vehicles/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(body),
      }),
    delete: (id: string) =>
      apiFetch<{ deleted: boolean; id: string }>(`/v1/vehicles/${id}`, { method: 'DELETE' }),
    deleteSystem: (vehicleId: string, systemKey: string) =>
      apiFetch<{ deleted: boolean; system: string }>(`/v1/vehicles/${vehicleId}/systems/${systemKey}`, { method: 'DELETE' }),
  },

  repairs: {
    get: (repairId: string) =>
      apiFetch<RepairDetail>(`/v1/repairs/${repairId}`),

    getNotes: (repairId: string) =>
      apiFetch<NoteWithAuthor[]>(`/v1/repairs/${repairId}/notes`),

    create: (body: { vehicle_id: string; title: string; system: string }) =>
      apiFetch<Repair>('/v1/repairs', {
        method: 'POST',
        body: JSON.stringify(body),
      }),

    updateStatus: (repairId: string, status: string) =>
      apiFetch<Repair>(`/v1/repairs/${repairId}`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      }),

    toggleStep: (repairId: string, stepId: string) =>
      apiFetch<{ step: RepairStep; repair: Repair }>(
        `/v1/repairs/${repairId}/steps/${stepId}/complete`,
        { method: 'POST' }
      ),

    createStep: (repairId: string, body: { position: number; title: string; notes?: string; torque_spec?: string }) =>
      apiFetch<RepairStep>(`/v1/repairs/${repairId}/steps`, {
        method: 'POST',
        body: JSON.stringify(body),
      }),

    deleteStep: (repairId: string, stepId: string) =>
      apiFetch<{ deleted: boolean; id: string }>(`/v1/repairs/${repairId}/steps/${stepId}`, {
        method: 'DELETE',
      }),

    createPart: (repairId: string, body: { name: string; part_number?: string; supplier?: string; cost?: string; notes?: string }) =>
      apiFetch<Part>(`/v1/repairs/${repairId}/parts`, {
        method: 'POST',
        body: JSON.stringify(body),
      }),

    updatePartStatus: (repairId: string, partId: string, status: string) =>
      apiFetch<Part>(`/v1/repairs/${repairId}/parts/${partId}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      }),

    createNote: (repairId: string, body: string) =>
      apiFetch<RepairNote>(`/v1/repairs/${repairId}/notes`, {
        method: 'POST',
        body: JSON.stringify({ body }),
      }),

    deleteRepair: (repairId: string) =>
      apiFetch<{ deleted: boolean; id: string }>(`/v1/repairs/${repairId}`, {
        method: 'DELETE',
      }),

    reopen: (repairId: string) =>
      apiFetch<Repair>(`/v1/repairs/${repairId}`, {
        method: 'PATCH',
        body: JSON.stringify({ status: 'in_progress' }),
      }),
  },
}

// Shim for existing call sites that still read `useApi()` as a hook.
// Session cookie carries auth, so this is a trivial pass-through.
export function useApi() {
  return api
}
