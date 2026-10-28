import { useEffect, useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { PAGE_SIZE } from '../../../api/customers';
import type { CustomerStatus } from '../../../api/types';

// The filter of status as it shows in the URL, which the user sees.
const statusParam = enumParam(['activos', 'inactivos', 'todos'], 'activos');
type StatusParam = typeof statusParam.fallback;
const statusOf: Record<StatusParam, CustomerStatus> = { activos: 'active', inactivos: 'inactive', todos: 'all' };
const paramOf: Record<CustomerStatus, StatusParam> = { active: 'activos', inactive: 'inactivos', all: 'todos' };

const textParam = stringParam();
// A new search or filter starts at the first page.
const resetPage = { reset: ['pagina'] };
const searchOptions = { ...resetPage, replace: true };

/** The search, segment, status and page of the customer list, kept in the URL (?q=&segmento=&estado=&pagina=). */
export function useCustomerListParams() {
  const [q, setQ] = useSearchParamsState('q', textParam, searchOptions);
  const [segment, setSegment] = useSearchParamsState('segmento', textParam, resetPage);
  const [status, setStatus] = useSearchParamsState('estado', statusParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  // The text is searched once the user stops typing: one request, not one per key.
  const [text, setText] = useState(q);
  const searched = useDebouncedValue(text.trim());
  useEffect(() => {
    if (searched !== q) setQ(searched);
  }, [searched, q, setQ]);

  const apiStatus = statusOf[status];
  return {
    q,
    /** The page of customers to ask the API for. */
    query: {
      q: q || undefined,
      segment_id: segment || undefined,
      status: apiStatus,
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
    },
    /** Whether a search or a filter is narrowing the list. */
    filtered: Boolean(q || segment) || apiStatus !== 'active',
    text,
    setText,
    segment,
    setSegment,
    status: apiStatus,
    setStatus: (next: CustomerStatus) => setStatus(paramOf[next]),
    offset: (page - 1) * PAGE_SIZE,
    setOffset: (offset: number) => setPage(offset / PAGE_SIZE + 1),
  };
}

export type CustomerListParams = ReturnType<typeof useCustomerListParams>;
