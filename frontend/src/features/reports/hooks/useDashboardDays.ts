import { useEffect, useMemo, useState } from 'react';
import { DASHBOARD_REFRESH_MS } from '@/shared/query';
import { dashboardDays } from '../model/dashboard/dashboardDays';

/**
 * The days of the dashboard, moved forward every few minutes: the cut of
 * today's sales is part of their query, so a new time reads them again.
 */
export function useDashboardDays() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const id = setInterval(() => {
      setNow(new Date());
    }, DASHBOARD_REFRESH_MS);
    return () => {
      clearInterval(id);
    };
  }, []);
  return useMemo(() => dashboardDays(now), [now]);
}
