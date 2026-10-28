import { useMutation, useQueryClient } from '@tanstack/react-query';
import { customersKeys, type CustomerSegment } from '@/features/customers';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

type SaveSegment = { id?: string; body: components['schemas']['CustomerSegmentInput'] };

/** Creates a segment, or renames it when it has an id; the segments reload before it resolves. */
export function useSaveCustomerSegment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, body }: SaveSegment) =>
      unwrap(
        id
          ? await api.PUT('/api/v1/customer-segments/{segmentId}', { params: { path: { segmentId: id } }, body })
          : await api.POST('/api/v1/customer-segments', { body }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: customersKeys.segments }),
  });
}

/** Deletes a segment without customers; the API answers customer_segment_in_use otherwise. */
export function useDeleteCustomerSegment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (segment: CustomerSegment) =>
      unwrapEmpty(
        await api.DELETE('/api/v1/customer-segments/{segmentId}', { params: { path: { segmentId: segment.id } } }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: customersKeys.segments }),
  });
}
