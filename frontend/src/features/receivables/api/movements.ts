import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { receivablesKeys } from './keys';
import type { PageQuery } from './types';

export const PAGE_SIZE = 20;

/** A page of the customer's statement, most recent first. While the next page loads, the previous one stays. */
export function useCreditMovements(customerId: string, query: PageQuery) {
  return useQuery({
    queryKey: receivablesKeys.movements(customerId, query),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/customers/{customerId}/credit/movements', {
          params: { path: { customerId }, query },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}
