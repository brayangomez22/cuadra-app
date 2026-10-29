import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { PURCHASE_PAGE_SIZE } from '../../../api/purchases';
import type { PurchaseStatusFilter } from '../../../api/types';

// The filter of status as it shows in the URL, which the user sees.
const statusParam = enumParam(['todas', 'por-pagar', 'pagadas'], 'todas');
type StatusParam = typeof statusParam.fallback;
const statusOf: Record<StatusParam, PurchaseStatusFilter> = { todas: 'all', 'por-pagar': 'open', pagadas: 'paid' };
const paramOf: Record<PurchaseStatusFilter, StatusParam> = { all: 'todas', open: 'por-pagar', paid: 'pagadas' };

const textParam = stringParam();
// A new filter starts at the first page.
const resetPage = { reset: ['pagina'] };

/** The supplier, status, dates and page of the purchase list, kept in the URL (?proveedor=&estado=&desde=&hasta=&pagina=). */
export function usePurchaseListParams() {
  const [supplier, setSupplier] = useSearchParamsState('proveedor', textParam, resetPage);
  const [status, setStatus] = useSearchParamsState('estado', statusParam, resetPage);
  const [from, setFrom] = useSearchParamsState('desde', textParam, resetPage);
  const [to, setTo] = useSearchParamsState('hasta', textParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  const apiStatus = statusOf[status];
  const offset = (page - 1) * PURCHASE_PAGE_SIZE;
  return {
    /** The page of purchases to ask the API for. */
    query: {
      supplier_id: supplier || undefined,
      status: apiStatus,
      from: from || undefined,
      to: to || undefined,
      limit: PURCHASE_PAGE_SIZE,
      offset,
    },
    /** Whether a filter is narrowing the list. */
    filtered: Boolean(supplier || from || to) || apiStatus !== 'all',
    supplier,
    setSupplier,
    status: apiStatus,
    setStatus: (next: PurchaseStatusFilter) => setStatus(paramOf[next]),
    from,
    setFrom,
    to,
    setTo,
    offset,
    setOffset: (next: number) => setPage(next / PURCHASE_PAGE_SIZE + 1),
  };
}

export type PurchaseListParams = ReturnType<typeof usePurchaseListParams>;
