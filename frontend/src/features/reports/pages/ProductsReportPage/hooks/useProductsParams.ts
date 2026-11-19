import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import type { ProductsReportSort } from '../../../api/types';

export const PRODUCTS_PAGE_SIZE = 20;

const sortParam = enumParam<ProductsReportSort>(['revenue', 'quantity', 'profit'], 'revenue');
const textParam = stringParam();
// A new filter starts at the first page.
const resetPage = { reset: ['pagina'] };

/** The products report's own filters and page, in the URL (?categoria=&atributo=&valor=&orden=&pagina=). */
export function useProductsParams() {
  const [category, setCategory] = useSearchParamsState('categoria', textParam, resetPage);
  // Another attribute starts without a value.
  const [attribute, setAttribute] = useSearchParamsState('atributo', textParam, { reset: ['pagina', 'valor'] });
  const [value, setValue] = useSearchParamsState('valor', textParam, resetPage);
  const [sort, setSort] = useSearchParamsState('orden', sortParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PRODUCTS_PAGE_SIZE;
  // The attribute filter goes only with both its key and its value.
  const byAttribute = attribute && value ? { attribute, attribute_value: value } : {};
  return {
    category,
    setCategory,
    attribute,
    setAttribute,
    value,
    setValue,
    sort,
    setSort: (next: string) => setSort(sortParam.parse(next) ?? 'revenue'),
    offset,
    setOffset: (next: number) => setPage(next / PRODUCTS_PAGE_SIZE + 1),
    query: { category_id: category || undefined, sort, ...byAttribute },
  };
}

export type ProductsParams = ReturnType<typeof useProductsParams>;
