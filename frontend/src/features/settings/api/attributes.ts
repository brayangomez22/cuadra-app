import { useMutation, useQueryClient } from '@tanstack/react-query';
import { catalogKeys, type AttributeDefinition } from '@/features/catalog';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];

/** The key and type are set when the attribute is created and never change. */
export type SaveAttribute =
  { body: Schemas['AttributeDefinitionInput'] } | { id: string; body: Schemas['AttributeDefinitionUpdate'] };

/** Creates or updates a product attribute; the attributes reload before it resolves. */
export function useSaveAttribute() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: SaveAttribute) =>
      'id' in input
        ? unwrap(
            await api.PUT('/api/v1/product-attributes/{attributeId}', {
              params: { path: { attributeId: input.id } },
              body: input.body,
            }),
          )
        : unwrap(await api.POST('/api/v1/product-attributes', { body: input.body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: catalogKeys.attributes }),
  });
}

export function useDeleteAttribute() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (attribute: AttributeDefinition) =>
      unwrapEmpty(
        await api.DELETE('/api/v1/product-attributes/{attributeId}', {
          params: { path: { attributeId: attribute.id } },
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: catalogKeys.attributes }),
  });
}
