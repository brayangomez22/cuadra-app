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

// The order as it shows in the URL; margin only for who sees costs.
const sortParam = enumParam(['nombre', 'precio', 'margen'], 'nombre');
const sortOf = { nombre: 'name', precio: 'price', margen: 'margin' } as const;
export type ProductSort = (typeof sortOf)[keyof typeof sortOf];
const sortParamOf: Record<ProductSort, typeof sortParam.fallback> = {
  name: 'nombre',
  price: 'precio',
  margin: 'margen',
};
const isSort = (value: string): value is ProductSort => value in sortParamOf;

const qParam = stringParam();
// A new search or filter starts at the first page.
const resetPage = { reset: ['pagina'] };
const searchOptions = { ...resetPage, replace: true };

/** The search, status, category, order and page of the product list, in the URL (?q=&estado=&categoria=&orden=&pagina=). */
export function useProductListParams() {
  const [q, setQ] = useSearchParamsState('q', qParam, searchOptions);
  const [status, setStatus] = useSearchParamsState('estado', statusParam, resetPage);
  const [category, setCategory] = useSearchParamsState('categoria', qParam, resetPage);
  const [sort, setSort] = useSearchParamsState('orden', sortParam, resetPage);
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
    category,
    setCategory,
    sort: sortOf[sort],
    setSort: (next: string) => setSort(isSort(next) ? sortParamOf[next] : 'nombre'),
    offset: (page - 1) * PAGE_SIZE,
    setOffset: (offset: number) => setPage(offset / PAGE_SIZE + 1),
  };
}

export type ProductListParams = ReturnType<typeof useProductListParams>;
