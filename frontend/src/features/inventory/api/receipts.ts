import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';
import { inventoryKeys } from './keys';

type ReceiptInput = components['schemas']['ReceiptInput'];

/** Registers goods that arrive at a location; every stock and kardex shown is stale after it. */
export function useReceiveGoods() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: ReceiptInput) => unwrap(await api.POST('/api/v1/inventory/receipts', { body })),
    // Without waiting for the refetch: the form is ready for the next product right away.
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: inventoryKeys.all }),
  });
}
