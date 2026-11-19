import { stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { todayInBogota } from '@/shared/format';
import { defaultPeriod, periodProblem } from '../model/period';

// A new period or location starts a report at its first page.
const resetPage = { reset: ['pagina'] };

/**
 * The period and location of a report, kept in the URL (?desde=&hasta=&sede=)
 * so a report can be shared and the other reports open on the same days.
 * Without them, this month so far and every location.
 */
export function useReportParams() {
  const today = defaultPeriod(todayInBogota());
  const [from, setFrom] = useSearchParamsState('desde', stringParam(today.from), resetPage);
  const [to, setTo] = useSearchParamsState('hasta', stringParam(today.to), resetPage);
  const [location, setLocation] = useSearchParamsState('sede', stringParam(), resetPage);
  const problem = periodProblem(from, to);
  return {
    from,
    setFrom,
    to,
    setTo,
    location,
    setLocation,
    /** Why the period cannot be asked for, or null. */
    problem,
    /** The filters of the API, for any report with a period. */
    query: { from, to, location_id: location || undefined },
    /** The query of the report links, so they keep the period and the location. */
    links: { desde: from, hasta: to, sede: location || undefined },
  };
}

export type ReportParams = ReturnType<typeof useReportParams>;
