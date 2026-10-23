// The alternate units a variant is sold and bought in (docs/decisiones.md,
// T26): each one is saved on its own with PUT or DELETE …/units/{unitId}, not
// with the product's form, because it belongs to a variant that must exist.
import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import { useStoreProduct } from './products';
import type { UnitOfMeasure, VariantUnitInput } from './types';

type UnitTarget = { productId: string; variantId: string; unit: UnitOfMeasure };

const unitPath = '/api/v1/products/{productId}/variants/{variantId}/units/{unitId}';
const pathOf = ({ productId, variantId, unit }: UnitTarget) => ({ path: { productId, variantId, unitId: unit.id } });

/** Adds or updates an alternate unit of a variant; its errors go to the form that calls it. */
export function useSaveVariantUnit() {
  const store = useStoreProduct();
  const { onSuccess } = useMutationToast({
    success: ({ unit }: { unit: UnitOfMeasure }) => ({ title: 'Unidad guardada', description: unit.name }),
    error: 'No se pudo guardar la unidad',
  });
  return useMutation({
    mutationFn: async ({ body, ...target }: UnitTarget & { body: VariantUnitInput }) => ({
      product: unwrap(await api.PUT(unitPath, { params: pathOf(target), body })),
      unit: target.unit,
    }),
    onSuccess: (saved) => {
      store(saved.product);
      onSuccess(saved);
    },
  });
}

/** Removes an alternate unit of a variant. */
export function useRemoveVariantUnit() {
  const store = useStoreProduct();
  const { onSuccess, onError } = useMutationToast({
    success: ({ unit }: { unit: UnitOfMeasure }) => ({ title: 'Unidad quitada', description: unit.name }),
    error: 'No se pudo quitar la unidad',
  });
  return useMutation({
    mutationFn: async (target: UnitTarget) => ({
      product: unwrap(await api.DELETE(unitPath, { params: pathOf(target) })),
      unit: target.unit,
    }),
    onSuccess: (removed) => {
      store(removed.product);
      onSuccess(removed);
    },
    onError,
  });
}
