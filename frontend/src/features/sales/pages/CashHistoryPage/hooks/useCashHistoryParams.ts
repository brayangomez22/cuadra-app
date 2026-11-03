import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { CASH_HISTORY_PAGE_SIZE } from '../../../api/cashSessions';
import type { CashSessionQuery } from '../../../api/types';

type Status = NonNullable<CashSessionQuery['status']>;

// The status as it shows in the URL, which the user sees.
const statusParam = enumParam(['todas', 'abiertas', 'cerradas'], 'todas');
const statusOf: Record<typeof statusParam.fallback, Status> = { todas: 'all', abiertas: 'open', cerradas: 'closed' };
const paramOf: Record<Status, typeof statusParam.fallback> = { all: 'todas', open: 'abiertas', closed: 'cerradas' };

const textParam = stringParam();
// A new filter starts at the first page.
const resetPage = { reset: ['pagina'] };

/** The status, location, days and page of the cash history, kept in the URL (?estado=&sede=&desde=&hasta=&pagina=). */
export function useCashHistoryParams() {
  const [status, setStatus] = useSearchParamsState('estado', statusParam, resetPage);
  const [location, setLocation] = useSearchParamsState('sede', textParam, resetPage);
  const [from, setFrom] = useSearchParamsState('desde', textParam, resetPage);
  const [to, setTo] = useSearchParamsState('hasta', textParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  const offset = (page - 1) * CASH_HISTORY_PAGE_SIZE;
  return {
    /** The page of drawers to ask the API for. */
    query: {
      status: statusOf[status],
      location_id: location || undefined,
      from: from || undefined,
      to: to || undefined,
      limit: CASH_HISTORY_PAGE_SIZE,
      offset,
    },
    status: statusOf[status],
    setStatus: (next: Status) => setStatus(paramOf[next]),
    location,
    setLocation,
    from,
    setFrom,
    to,
    setTo,
    offset,
    setOffset: (next: number) => setPage(next / CASH_HISTORY_PAGE_SIZE + 1),
  };
}

export type CashHistoryParams = ReturnType<typeof useCashHistoryParams>;
