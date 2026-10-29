import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { inventoryKeys } from '@/features/inventory';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { purchasesKeys } from './keys';
import type { PurchaseInput, PurchaseQuery } from './types';

export const PURCHASE_PAGE_SIZE = 20;

/** A page of purchases, the latest invoice first. While the next page loads, the previous one stays on screen. */
export function usePurchases(query: PurchaseQuery) {
  return useQuery({
    queryKey: purchasesKeys.purchaseList(query),
    queryFn: async () => unwrap(await api.GET('/api/v1/purchases', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

export function usePurchase(id: string | undefined) {
  return useQuery({
    queryKey: purchasesKeys.purchase(id ?? ''),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/purchases/{purchaseId}', { params: { path: { purchaseId: id ?? '' } } })),
    enabled: Boolean(id),
  });
}

/**
 * Registers the purchase. Its lines enter the stock and change the average
 * cost, and on credit the supplier owes more: the purchases, the suppliers
 * and the inventory are asked again.
 */
export function useRegisterPurchase() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: PurchaseInput) => unwrap(await api.POST('/api/v1/purchases', { body })),
    onSuccess: (purchase) => {
      queryClient.setQueryData(purchasesKeys.purchase(purchase.id), purchase);
      void queryClient.invalidateQueries({ queryKey: purchasesKeys.purchases });
      void queryClient.invalidateQueries({ queryKey: purchasesKeys.suppliers });
      void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
    },
  });
}
