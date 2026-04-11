import { useQuery } from '@tanstack/react-query'
import { useApi } from './api'

export function useGarage() {
    const api = useApi()
    return useQuery({
        queryKey: ['garage'],
        queryFn: api.garage.get,
    })
}