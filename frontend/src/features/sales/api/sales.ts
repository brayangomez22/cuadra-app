import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { inventoryKeys } from '@/features/inventory';
import { receivablesKeys } from '@/features/receivables';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { salesKeys } from './keys';
import type { ConfirmSaleInput } from './types';

export const SALE_PAGE_SIZE = 20;

type SaleQuery = { date: string; location_id?: string; limit: number; offset: number };

/** A page of the sales of a day, newest first. */
export function useSales(query: SaleQuery) {
  return useQuery({
    queryKey: salesKeys.saleList(query),
    queryFn: async () => unwrap(await api.GET('/api/v1/sales', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

export function useSale(id: string) {
  return useQuery({
    queryKey: salesKeys.sale(id),
    queryFn: async () => unwrap(await api.GET('/api/v1/sales/{saleId}', { params: { path: { saleId: id } } })),
  });
}

/**
 * Voids the sale: its stock comes back and a credit sale cancels its debt, so
 * those screens read again. A cash session stops counting it. A refused void
 * reads the sale again: someone else may have voided it.
 */
export function useVoidSale(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (reason: string) =>
      unwrap(await api.POST('/api/v1/sales/{saleId}/void', { params: { path: { saleId: id } }, body: { reason } })),
    onSuccess: (sale) => {
      queryClient.setQueryData(salesKeys.sale(id), sale);
      void queryClient.invalidateQueries({ queryKey: salesKeys.sales });
      void queryClient.invalidateQueries({ queryKey: salesKeys.cashSessions });
      void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
      if (sale.customer_id)
        void queryClient.invalidateQueries({ queryKey: receivablesKeys.customer(sale.customer_id) });
      void queryClient.invalidateQueries({ queryKey: receivablesKeys.agings });
    },
    onError: () => queryClient.invalidateQueries({ queryKey: salesKeys.sale(id) }),
  });
}

/** How much the user may discount on a line. The API applies it again on checkout. */
export function useDiscountLimit() {
  return useQuery({
    queryKey: salesKeys.discountLimit,
    queryFn: async () => unwrap(await api.GET('/api/v1/sales/discount-limit')),
  });
}

export function useConfirmSale() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: ConfirmSaleInput) => unwrap(await api.POST('/api/v1/sales', { body })),
    // The sale took stock, and a credit sale the customer's credit: their
    // screens read them again. Not awaited: the cashier sees the sale without
    // waiting for those refetches.
    onSuccess: (sale) => {
      void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
      if (sale.customer_id)
        void queryClient.invalidateQueries({ queryKey: receivablesKeys.customer(sale.customer_id) });
      void queryClient.invalidateQueries({ queryKey: receivablesKeys.agings });
    },
  });
}
