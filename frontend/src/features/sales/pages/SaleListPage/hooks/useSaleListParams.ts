import { todayInBogota } from '@/shared/format';
import { pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { SALE_PAGE_SIZE } from '../../../api/sales';

const textParam = stringParam();
// A new filter starts at the first page.
const resetPage = { reset: ['pagina'] };

/** The day, location and page of the sales list, kept in the URL (?fecha=&sede=&pagina=). Without a day, today. */
export function useSaleListParams() {
  const [date, setDate] = useSearchParamsState('fecha', textParam, resetPage);
  const [location, setLocation] = useSearchParamsState('sede', textParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  const day = date || todayInBogota();
  const offset = (page - 1) * SALE_PAGE_SIZE;
  return {
    /** The page of sales to ask the API for. */
    query: { date: day, location_id: location || undefined, limit: SALE_PAGE_SIZE, offset },
    day,
    setDay: setDate,
    location,
    setLocation,
    offset,
    setOffset: (next: number) => setPage(next / SALE_PAGE_SIZE + 1),
  };
}

export type SaleListParams = ReturnType<typeof useSaleListParams>;
