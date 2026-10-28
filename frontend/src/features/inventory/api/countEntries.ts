import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import type { components } from '@/shared/api/schema';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import { inventoryKeys } from './keys';

// The latest entries the counting screen lists; the API pages them.
const SHOWN = 50;

/** What the user counted (all of it for someone who manages counts), newest first. */
export function useCountEntries(countId: string) {
  return useQuery({
    queryKey: inventoryKeys.countEntries(countId),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/stock-counts/{stockCountId}/entries', {
          params: { path: { stockCountId: countId }, query: { limit: SHOWN } },
        }),
      ),
  });
}

// An entry changes the progress and the differences of the count.
function useInvalidateCount(countId: string) {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: inventoryKeys.stockCount(countId) });
}

/** Records what the user counted of a variant. */
export function useRecordCountEntry(countId: string) {
  const invalidate = useInvalidateCount(countId);
  return useMutation({
    mutationFn: async (body: components['schemas']['CountEntryInput']) =>
      unwrap(
        await api.POST('/api/v1/stock-counts/{stockCountId}/entries', {
          params: { path: { stockCountId: countId } },
          body,
        }),
      ),
    onSuccess: invalidate,
  });
}

/** Deletes something counted by mistake. */
export function useDeleteCountEntry(countId: string) {
  const invalidate = useInvalidateCount(countId);
  return useMutation({
    mutationFn: async (entryId: string) =>
      unwrapEmpty(
        await api.DELETE('/api/v1/stock-counts/{stockCountId}/entries/{countEntryId}', {
          params: { path: { stockCountId: countId, countEntryId: entryId } },
        }),
      ),
    onSuccess: invalidate,
  });
}
