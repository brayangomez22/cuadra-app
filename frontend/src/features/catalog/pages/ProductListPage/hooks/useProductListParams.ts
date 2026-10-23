import { useEffect, useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { PAGE_SIZE } from '../../../api/products';
import type { ProductStatus } from '../../../api/types';

// The filter of status as it shows in the URL, which the user sees.
const statusParam = enumParam(['activos', 'inactivos', 'todos'], 'activos');
type StatusParam = typeof statusParam.fallback;
const statusOf: Record<StatusParam, ProductStatus> = { activos: 'active', inactivos: 'inactive', todos: 'all' };
const paramOf: Record<ProductStatus, StatusParam> = { active: 'activos', inactive: 'inactivos', all: 'todos' };

const qParam = stringParam();
// A new search or filter starts at the first page.
const resetPage = { reset: ['pagina'] };
const searchOptions = { ...resetPage, replace: true };

/** The search, status and page of the product list, kept in the URL (?q=&estado=&pagina=). */
export function useProductListParams() {
  const [q, setQ] = useSearchParamsState('q', qParam, searchOptions);
  const [status, setStatus] = useSearchParamsState('estado', statusParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  // The text is searched once the user stops typing: one request, not one per key.
  const [text, setText] = useState(q);
  const searched = useDebouncedValue(text.trim());
  useEffect(() => {
    if (searched !== q) setQ(searched);
  }, [searched, q, setQ]);

  return {
    q,
    text,
    setText,
    status: statusOf[status],
    setStatus: (next: ProductStatus) => setStatus(paramOf[next]),
    offset: (page - 1) * PAGE_SIZE,
    setOffset: (offset: number) => setPage(offset / PAGE_SIZE + 1),
  };
}
