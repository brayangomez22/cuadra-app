import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { DASHBOARD_REFRESH_MS } from '@/shared/query';
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

// The totals add up every customer, whatever the page: one row is enough.
const totalsQuery = { limit: 1, offset: 0 };

/** What every customer owes by age, for the home screen: read again every few minutes. */
export function useAgingTotals({ enabled = true } = {}) {
  return useQuery({
    queryKey: receivablesKeys.aging(totalsQuery),
    queryFn: async () => unwrap(await api.GET('/api/v1/receivables/aging', { params: { query: totalsQuery } })),
    select: (aging) => aging.totals,
    enabled,
    refetchInterval: DASHBOARD_REFRESH_MS,
  });
}
