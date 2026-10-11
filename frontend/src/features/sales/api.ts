import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

export type Sale = components['schemas']['Sale'];
export type ConfirmSaleInput = components['schemas']['ConfirmSaleInput'];

export const salesKeys = {
  discountLimit: ['sales', 'discount-limit'] as const,
};

/** How much the user may discount on a line. The API applies it again on checkout. */
export function useDiscountLimit() {
  return useQuery({
    queryKey: salesKeys.discountLimit,
    queryFn: async () => unwrap(await api.GET('/api/v1/sales/discount-limit')),
  });
}

export function useConfirmSale() {
  return useMutation({
    mutationFn: async (body: ConfirmSaleInput) => unwrap(await api.POST('/api/v1/sales', { body })),
  });
}
