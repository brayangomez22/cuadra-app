import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { purchasesKeys } from './keys';
import type { SupplierPaymentInput } from './types';

export const PAYABLES_PAGE_SIZE = 20;

/** The payments of a purchase, the first one first; only a purchase on credit has them. */
export function useSupplierPayments(purchaseId: string, { enabled }: { enabled: boolean }) {
  return useQuery({
    queryKey: purchasesKeys.payments(purchaseId),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/purchases/{purchaseId}/payments', { params: { path: { purchaseId } } })),
    enabled,
  });
}

/** Records a payment to the supplier: the purchase, its payments, the lists and what is owed change. */
export function useRecordSupplierPayment(purchaseId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: SupplierPaymentInput) =>
      unwrap(await api.POST('/api/v1/purchases/{purchaseId}/payments', { params: { path: { purchaseId } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: purchasesKeys.purchases });
      void queryClient.invalidateQueries({ queryKey: purchasesKeys.suppliers });
      void queryClient.invalidateQueries({ queryKey: purchasesKeys.payables });
    },
  });
}

/** A page of what is owed to each supplier, with the totals of all of them. */
export function usePayables(offset: number) {
  return useQuery({
    queryKey: purchasesKeys.payablesPage(offset),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/payables', { params: { query: { limit: PAYABLES_PAGE_SIZE, offset } } })),
    placeholderData: keepPreviousData,
  });
}
