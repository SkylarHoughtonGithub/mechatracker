import { useQuery } from '@tanstack/react-query'
import { useApi } from './api'

export function useGarage() {
    const api = useApi()
    return useQuery({
        queryKey: ['garage'],
        queryFn: api.garage.get,
    })
}

export function useVehicle(id: string) {
    const api = useApi()
    return useQuery({
        queryKey: ['vehicle', id],
        queryFn: () => api.vehicles.get(id),
        enabled: !!id,
    })
}

export function useVehicleRepairs(vehicleId: string, system?: string) {
    const api = useApi()
    return useQuery({
        queryKey: ['vehicle-repairs', vehicleId, system],
        queryFn: () => api.vehicles.repairs(vehicleId, system),
        enabled: !!vehicleId,
    })
}

export function useRepair(repairId: string) {
    const api = useApi()
    return useQuery({
        queryKey: ['repair', repairId],
        queryFn: () => api.repairs.get(repairId),
        enabled: !!repairId,
    })
}

export function useNotes(repairId: string) {
    const api = useApi()
    return useQuery({
        queryKey: ['repair-notes', repairId],
        queryFn: () => api.repairs.getNotes(repairId),
        enabled: !!repairId,
    })
}

export function useVehicleSystemLabel(vehicleId: string, systemId: string): string {
    const { data: garage } = useGarage()
    const vehicle = garage?.vehicles?.find(v => v.id === vehicleId)
    return vehicle?.systems?.find(s => s.id === systemId)?.label ?? systemId
}