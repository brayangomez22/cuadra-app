import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';
import { inventoryKeys } from './keys';

type AdjustmentInput = components['schemas']['AdjustmentInput'];
type TransferInput = components['schemas']['TransferInput'];

/** Corrects a location's stock; every stock and kardex shown is stale after it. */
export function useAdjustStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: AdjustmentInput) => unwrap(await api.POST('/api/v1/inventory/adjustments', { body })),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: inventoryKeys.all }),
  });
}

/** Moves goods between two locations; every stock and kardex shown is stale after it. */
export function useTransferStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: TransferInput) => unwrap(await api.POST('/api/v1/inventory/transfers', { body })),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: inventoryKeys.all }),
  });
}
