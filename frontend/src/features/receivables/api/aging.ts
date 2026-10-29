import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { receivablesKeys } from './keys';
import type { PageQuery } from './types';

export const PAGE_SIZE = 20;

/** A page of the receivables by age, the customers with most overdue first. */
export function useAging(query: PageQuery) {
  return useQuery({
    queryKey: receivablesKeys.aging(query),
    queryFn: async () => unwrap(await api.GET('/api/v1/receivables/aging', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}
