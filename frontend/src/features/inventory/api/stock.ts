import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { inventoryKeys } from './keys';
import { PAGE_SIZE } from './types';

/**
 * A page of a location's stock; with variantIds, only theirs (the ones a
 * search found). An empty list asks for nothing: the whole location is not
 * what nothing matching means.
 */
export function useStock(locationId: string | undefined, offset: number, variantIds?: string[]) {
  return useQuery({
    queryKey: inventoryKeys.stock(locationId ?? '', offset, variantIds),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/locations/{locationId}/stock', {
          params: {
            path: { locationId: locationId ?? '' },
            query: { limit: PAGE_SIZE, offset, variant_ids: variantIds },
          },
        }),
      ),
    enabled: Boolean(locationId) && variantIds?.length !== 0,
    placeholderData: keepPreviousData,
  });
}
