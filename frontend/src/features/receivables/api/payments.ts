import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { receivablesKeys } from './keys';
import type { CustomerPaymentInput } from './types';

/** Records a payment of the customer; their credit, statement and the aging are asked again. */
export function useRecordPayment(customerId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CustomerPaymentInput) =>
      unwrap(await api.POST('/api/v1/customers/{customerId}/payments', { params: { path: { customerId } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: receivablesKeys.customer(customerId) });
      void queryClient.invalidateQueries({ queryKey: receivablesKeys.agings });
    },
  });
}
