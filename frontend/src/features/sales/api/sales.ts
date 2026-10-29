import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { inventoryKeys } from '@/features/inventory';
import { receivablesKeys } from '@/features/receivables';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { salesKeys } from './keys';
import type { ConfirmSaleInput } from './types';

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
