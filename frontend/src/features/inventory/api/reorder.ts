import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import { DASHBOARD_REFRESH_MS } from '@/shared/query';
import { inventoryKeys } from './keys';
import { PAGE_SIZE, type ReorderSort } from './types';

type AlertsQuery = {
  locationId?: string;
  offset?: number;
  limit?: number;
  sort?: ReorderSort;
  enabled?: boolean;
  /** Read again every few minutes, for the home screen. */
  live?: boolean;
};

/** The variants at or below their minimum: of one location, or of all without locationId. */
export function useReorderAlerts({ enabled = true, live = false, ...rest }: AlertsQuery) {
  const query = { offset: 0, limit: PAGE_SIZE, sort: 'urgency' as const, ...rest };
  return useQuery({
    queryKey: inventoryKeys.reorderAlerts(query),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/inventory/reorder-alerts', {
          params: {
            query: {
              location_id: query.locationId,
              limit: query.limit,
              offset: query.offset,
              sort: query.sort === 'urgency' ? undefined : query.sort,
            },
          },
        }),
      ),
    enabled,
    placeholderData: keepPreviousData,
    refetchInterval: live ? DASHBOARD_REFRESH_MS : false,
  });
}

type RuleTarget = { locationId: string; variantId: string };
type RuleBody = { min_quantity: string; max_quantity: string };

function rulePath({ locationId, variantId }: RuleTarget) {
  return { params: { path: { locationId, variantId } } };
}

// The stock rows show the rule and the alerts depend on it: both are read again.
function useInvalidateStock() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
}

/** Sets a variant's minimum and maximum at a location, replacing the previous ones. */
export function useSetReorderRule(target: RuleTarget) {
  const invalidate = useInvalidateStock();
  return useMutation({
    mutationFn: async (body: RuleBody) =>
      unwrap(await api.PUT('/api/v1/locations/{locationId}/reorder-rules/{variantId}', { ...rulePath(target), body })),
    onSuccess: invalidate,
  });
}

/** Removes a variant's minimum and maximum at a location. */
export function useDeleteReorderRule(target: RuleTarget) {
  const invalidate = useInvalidateStock();
  return useMutation({
    mutationFn: async () =>
      unwrapEmpty(await api.DELETE('/api/v1/locations/{locationId}/reorder-rules/{variantId}', rulePath(target))),
    onSuccess: invalidate,
  });
}
