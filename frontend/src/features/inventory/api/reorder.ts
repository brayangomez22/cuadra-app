import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { inventoryKeys } from './keys';
import { PAGE_SIZE } from './types';

type AlertsQuery = { locationId?: string; offset?: number; limit?: number; enabled?: boolean };

/** The variants at or below their minimum: of one location, or of all without locationId. */
export function useReorderAlerts({ locationId, offset = 0, limit = PAGE_SIZE, enabled = true }: AlertsQuery) {
  return useQuery({
    queryKey: inventoryKeys.reorderAlerts(locationId, offset, limit),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/inventory/reorder-alerts', {
          params: { query: { location_id: locationId, limit, offset } },
        }),
      ),
    enabled,
    placeholderData: keepPreviousData,
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
    // A 204 has no body (data is undefined): only an error status throws.
    mutationFn: async () => {
      const result = await api.DELETE('/api/v1/locations/{locationId}/reorder-rules/{variantId}', rulePath(target));
      if (!result.response.ok) unwrap(result);
    },
    onSuccess: invalidate,
  });
}
