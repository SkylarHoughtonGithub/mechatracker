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