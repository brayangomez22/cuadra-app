import { useState } from 'react';
import { pageParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { useCountLines } from '../../../api/stockCountReview';
import { PAGE_SIZE } from '../../../api/types';
import { useRowProducts } from '../../../hooks/useRowProducts';
import { lineFilterParam, toLineStatus } from '../../../model/stockCount/lineFilter';
import { rowVariant } from '../../../model/rowVariant';

// Another filter starts on its first page.
const resetPage = { reset: ['pagina'] };

/**
 * The lines of a count by the status in the URL (?estado=, ?pagina=), with
 * the products they belong to. marked is the line sent to recount from the
 * approval's error, which the screen shows among those under review.
 */
export function useCountReview(countId: string) {
  const [filter, setFilter] = useSearchParamsState('estado', lineFilterParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PAGE_SIZE;
  const lines = useCountLines(countId, toLineStatus(filter), offset);
  const rows = lines.data?.items ?? [];
  const { products, productsById } = useRowProducts(rows, lines.isPlaceholderData);
  const [marked, setMarked] = useState<string>();
  const nameOf = (variantId: string) => {
    const row = rows.find((r) => r.variant_id === variantId);
    return row && rowVariant(productsById.get(row.product_id), variantId).name;
  };
  const showRecount = (variantId: string) => {
    setMarked(variantId);
    setFilter('revision');
  };
  return {
    filter,
    setFilter,
    offset,
    setOffset: (next: number) => setPage(next / PAGE_SIZE + 1),
    lines,
    rows,
    products,
    productsById,
    marked,
    nameOf,
    showRecount,
  };
}
