import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import type { components } from '@/shared/api/schema';
import { unwrap } from '@/shared/api/errors';
import { inventoryKeys } from './keys';
import { PAGE_SIZE } from './types';

/** One page of the counts of a location, newest first; the approved ones bring their accuracy. */
export function useStockCounts(locationId: string, offset: number) {
  return useQuery({
    queryKey: inventoryKeys.stockCountList(locationId, offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/stock-counts', {
          params: { query: { location_id: locationId, limit: PAGE_SIZE, offset } },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

/** A count with how far it went; never what is expected (the count is blind). */
export function useStockCount(countId: string) {
  return useQuery({
    queryKey: inventoryKeys.stockCount(countId),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/stock-counts/{stockCountId}', { params: { path: { stockCountId: countId } } })),
  });
}

/** Opens a count at a location and takes the snapshot of its stock. */
export function useStartStockCount() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components['schemas']['StockCountInput']) =>
      unwrap(await api.POST('/api/v1/stock-counts', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: inventoryKeys.stockCounts }),
  });
}
