import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { inventoryKeys } from './keys';
import { PAGE_SIZE } from './types';

export function useStock(locationId: string | undefined, offset: number) {
  return useQuery({
    queryKey: inventoryKeys.stock(locationId ?? '', offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/locations/{locationId}/stock', {
          params: { path: { locationId: locationId ?? '' }, query: { limit: PAGE_SIZE, offset } },
        }),
      ),
    enabled: Boolean(locationId),
    placeholderData: keepPreviousData,
  });
}
