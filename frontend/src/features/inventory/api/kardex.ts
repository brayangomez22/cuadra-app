import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { inventoryKeys } from './keys';
import { PAGE_SIZE } from './types';

/** The kardex of a variant; it waits until the variant is known. */
export function useKardex(variantId: string | undefined, locationId: string | undefined, offset: number) {
  return useQuery({
    queryKey: inventoryKeys.kardex(variantId ?? '', locationId, offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/variants/{variantId}/kardex', {
          params: {
            path: { variantId: variantId ?? '' },
            query: { location_id: locationId, limit: PAGE_SIZE, offset },
          },
        }),
      ),
    enabled: Boolean(variantId),
    placeholderData: keepPreviousData,
  });
}
