import { useEffect, useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { SUPPLIER_PAGE_SIZE } from '../../../api/suppliers';
import type { SupplierStatus } from '../../../api/types';

// The filter of status as it shows in the URL, which the user sees.
const statusParam = enumParam(['activos', 'inactivos', 'todos'], 'activos');
type StatusParam = typeof statusParam.fallback;
const statusOf: Record<StatusParam, SupplierStatus> = { activos: 'active', inactivos: 'inactive', todos: 'all' };
const paramOf: Record<SupplierStatus, StatusParam> = { active: 'activos', inactive: 'inactivos', all: 'todos' };

const textParam = stringParam();
// A new search or filter starts at the first page.
const resetPage = { reset: ['pagina'] };

/** The search, status and page of the supplier list, kept in the URL (?q=&estado=&pagina=). */
export function useSupplierListParams() {
  const [q, setQ] = useSearchParamsState('q', textParam, { ...resetPage, replace: true });
  const [status, setStatus] = useSearchParamsState('estado', statusParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  // The text is searched once the user stops typing: one request, not one per key.
  const [text, setText] = useState(q);
  const searched = useDebouncedValue(text.trim());
  useEffect(() => {
    if (searched !== q) setQ(searched);
  }, [searched, q, setQ]);

  const apiStatus = statusOf[status];
  const offset = (page - 1) * SUPPLIER_PAGE_SIZE;
  return {
    q,
    /** The page of suppliers to ask the API for. */
    query: { q: q || undefined, status: apiStatus, limit: SUPPLIER_PAGE_SIZE, offset },
    /** Whether a search or a filter is narrowing the list. */
    filtered: Boolean(q) || apiStatus !== 'active',
    text,
    setText,
    status: apiStatus,
    setStatus: (next: SupplierStatus) => setStatus(paramOf[next]),
    offset,
    setOffset: (next: number) => setPage(next / SUPPLIER_PAGE_SIZE + 1),
  };
}

export type SupplierListParams = ReturnType<typeof useSupplierListParams>;
