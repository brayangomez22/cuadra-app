import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import { inventoryKeys } from './keys';
import { PAGE_SIZE, type StockCountLineStatus } from './types';

const countPath = (countId: string) => ({ params: { path: { stockCountId: countId } } });

/** One page of the evaluated lines of a count, those under review first and then the largest differences. */
export function useCountLines(countId: string, status: StockCountLineStatus | undefined, offset: number) {
  return useQuery({
    queryKey: inventoryKeys.countLines(countId, status, offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/stock-counts/{stockCountId}/lines', {
          params: { path: { stockCountId: countId }, query: { status, limit: PAGE_SIZE, offset } },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

/** Asks to count again the doubtful variants: what was counted of them no longer counts. */
export function useRequestRecount(countId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (variantIds: string[]) =>
      unwrapEmpty(
        await api.POST('/api/v1/stock-counts/{stockCountId}/recount', {
          ...countPath(countId),
          body: { variant_ids: variantIds },
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: inventoryKeys.stockCount(countId) }),
  });
}

// Approving adjusts the stock: every inventory screen is read again.
function useCloseCount(close: (countId: string) => Promise<unknown>, countId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => close(countId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: inventoryKeys.all }),
  });
}

/** Closes a count and records an adjustment per variant with a difference. */
export function useApproveStockCount(countId: string) {
  return useCloseCount(
    async (id) => unwrap(await api.POST('/api/v1/stock-counts/{stockCountId}/approve', countPath(id))),
    countId,
  );
}

/** Closes a count without adjusting the stock. */
export function useCancelStockCount(countId: string) {
  return useCloseCount(
    async (id) => unwrap(await api.POST('/api/v1/stock-counts/{stockCountId}/cancel', countPath(id))),
    countId,
  );
}
