import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { customersKeys } from './keys';
import type { Customer, CustomerInput, CustomerQuery } from './types';

export const PAGE_SIZE = 20;

/** A page of customers. While the next page loads, the previous one stays on screen. */
export function useCustomers(query: CustomerQuery, { enabled = true } = {}) {
  return useQuery({
    queryKey: customersKeys.list(query),
    queryFn: async () => unwrap(await api.GET('/api/v1/customers', { params: { query } })),
    placeholderData: keepPreviousData,
    enabled,
  });
}

export function useCustomer(id: string | undefined) {
  return useQuery({
    queryKey: customersKeys.customer(id ?? ''),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/customers/{customerId}', { params: { path: { customerId: id ?? '' } } })),
    enabled: Boolean(id),
  });
}

/** Creates the customer, or updates it when it exists (current); the lists are asked again. */
export function useSaveCustomer(current?: Customer) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CustomerInput) =>
      unwrap(
        current
          ? await api.PUT('/api/v1/customers/{customerId}', { params: { path: { customerId: current.id } }, body })
          : await api.POST('/api/v1/customers', { body }),
      ),
    onSuccess: (saved) => {
      queryClient.setQueryData(customersKeys.customer(saved.id), saved);
      void queryClient.invalidateQueries({ queryKey: customersKeys.lists });
    },
  });
}
