import { useMutation, useQueryClient } from '@tanstack/react-query';
import { catalogKeys, type UnitOfMeasure } from '@/features/catalog';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

type UnitInput = components['schemas']['UnitInput'];

/** Creates a unit, or updates it with its id; the units list reloads before it resolves. */
export function useSaveUnit() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, body }: { id?: string; body: UnitInput }) =>
      id
        ? unwrap(await api.PUT('/api/v1/units/{unitId}', { params: { path: { unitId: id } }, body }))
        : unwrap(await api.POST('/api/v1/units', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: catalogKeys.units }),
  });
}

export function useDeleteUnit() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (unit: UnitOfMeasure) =>
      unwrap(await api.DELETE('/api/v1/units/{unitId}', { params: { path: { unitId: unit.id } } })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: catalogKeys.units }),
  });
}
