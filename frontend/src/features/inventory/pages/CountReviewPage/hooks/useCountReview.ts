import { pageParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { useCountLines } from '../../../api/stockCountReview';
import { PAGE_SIZE } from '../../../api/types';
import { useRowProducts } from '../../../hooks/useRowProducts';
import { lineFilterParam, toLineStatus } from '../../../model/stockCount/lineFilter';

// Another filter starts on its first page.
const resetPage = { reset: ['pagina'] };

/** The lines of a count by the status in the URL (?estado=, ?pagina=), with the products they belong to. */
export function useCountReview(countId: string) {
  const [filter, setFilter] = useSearchParamsState('estado', lineFilterParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PAGE_SIZE;
  const lines = useCountLines(countId, toLineStatus(filter), offset);
  const rows = lines.data?.items ?? [];
  const { products, productsById } = useRowProducts(rows, lines.isPlaceholderData);
  return {
    filter,
    setFilter,
    offset,
    setOffset: (next: number) => setPage(next / PAGE_SIZE + 1),
    lines,
    rows,
    products,
    productsById,
  };
}
