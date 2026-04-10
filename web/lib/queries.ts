import { useQuery } from '@tanstack/react-query'
import { api } from './api'

export function useGarage() {
    return useQuery({
        queryKey: ['garage'],
        queryFn: api.garage.get,
    })
}