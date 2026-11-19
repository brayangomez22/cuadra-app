import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { DASHBOARD_REFRESH_MS } from '@/shared/query';
import { reportsKeys } from './keys';
import type { TeamControlQuery } from './reports';

/**
 * The team's control for the dashboard's alerts: the report's own query
 * (and cache), read again every few minutes while the home screen is open.
 */
export function useTeamWeeks(query: TeamControlQuery) {
  return useQuery({
    queryKey: reportsKeys.report('team_control', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/team-control', { params: { query } })),
    placeholderData: keepPreviousData,
    refetchInterval: DASHBOARD_REFRESH_MS,
  });
}
