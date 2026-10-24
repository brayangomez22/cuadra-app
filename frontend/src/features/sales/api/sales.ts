import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { inventoryKeys } from '@/features/inventory';
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
    // The sale took stock: the inventory screens read it again. Not awaited:
    // the cashier sees the change without waiting for that refetch.
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: inventoryKeys.all }),
  });
}
