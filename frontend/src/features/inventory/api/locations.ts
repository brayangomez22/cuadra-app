import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { inventoryKeys } from './keys';

export function useLocations() {
  return useQuery({
    queryKey: inventoryKeys.locations,
    queryFn: async () => unwrap(await api.GET('/api/v1/locations')).items,
  });
}

/** Creates a location (sede o bodega) and reloads the list before it resolves. */
export function useCreateLocation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: { name: string }) => unwrap(await api.POST('/api/v1/locations', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: inventoryKeys.locations }),
  });
}
