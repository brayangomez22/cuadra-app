import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { receivablesKeys } from './keys';
import type { CreditTermsInput } from './types';

const path = (customerId: string) => ({ params: { path: { customerId } } });

/** The credit of a customer: limit, term, what they owe and their open receivables. Off without a customer. */
export function useCustomerCredit(customerId: string | undefined, { enabled = true } = {}) {
  return useQuery({
    queryKey: receivablesKeys.credit(customerId ?? ''),
    queryFn: async () => unwrap(await api.GET('/api/v1/customers/{customerId}/credit', path(customerId ?? ''))),
    enabled: enabled && Boolean(customerId),
  });
}

/** Sets the customer's credit limit and term; the credit shows the new figures at once. */
export function useSetCustomerCredit(customerId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreditTermsInput) =>
      unwrap(await api.PUT('/api/v1/customers/{customerId}/credit', { ...path(customerId), body })),
    onSuccess: (credit) => {
      queryClient.setQueryData(receivablesKeys.credit(customerId), credit);
      void queryClient.invalidateQueries({ queryKey: receivablesKeys.agings });
    },
  });
}
