import { useMutation, useQueryClient } from '@tanstack/react-query';
import { catalogKeys, type VariantAxis } from '@/features/catalog';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];

/** An axis is created with its name; afterwards only its values change. */
export type SaveAxis = { body: Schemas['VariantAxisInput'] } | { id: string; body: Schemas['VariantAxisUpdate'] };

/** Creates or updates a variant axis; the axes reload before it resolves. */
export function useSaveVariantAxis() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: SaveAxis) =>
      'id' in input
        ? unwrap(
            await api.PUT('/api/v1/variant-axes/{axisId}', {
              params: { path: { axisId: input.id } },
              body: input.body,
            }),
          )
        : unwrap(await api.POST('/api/v1/variant-axes', { body: input.body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: catalogKeys.variantAxes }),
  });
}

export function useDeleteVariantAxis() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (axis: VariantAxis) =>
      unwrapEmpty(await api.DELETE('/api/v1/variant-axes/{axisId}', { params: { path: { axisId: axis.id } } })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: catalogKeys.variantAxes }),
  });
}
