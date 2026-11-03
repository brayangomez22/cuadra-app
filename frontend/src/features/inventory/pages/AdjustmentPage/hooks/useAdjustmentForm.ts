import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { Sellable } from '@/features/catalog';
import { ApiError } from '@/shared/api/errors';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useAdjustStock } from '../../../api/adjustments';
import {
  adjustmentErrorFields,
  adjustmentSchema,
  type AdjustmentFormInput,
  type AdjustmentValues,
} from '../../../model/stockChange/adjustmentSchema';
import { adjustedMessage, insufficientStockMessage } from '../../../model/stockChange/stockChangeMessages';
import { useAdjustmentDelta } from './useAdjustmentDelta';

export type AdjustmentSetup = {
  locations: { value: string; label: string }[];
  initialItem: Sellable | null;
  initialLocation: string;
};

/** The adjustment form; after one it is ready for the next product, in the same location. */
export function useAdjustmentForm({ locations, initialItem, initialLocation }: AdjustmentSetup) {
  const { toast } = useToast();
  const adjust = useAdjustStock();
  const form = useForm<AdjustmentFormInput, unknown, AdjustmentValues>({
    resolver: zodResolver(adjustmentSchema),
    defaultValues: { item: initialItem, location_id: initialLocation, direction: '', quantity: '', reason: '' },
  });
  const { current, deltaOf } = useAdjustmentDelta(form.control);
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, adjustmentErrorFields);

  const submit = form.handleSubmit((values) => {
    const { item, location_id, reason } = values;
    clearFormError();
    const change = deltaOf(values);
    if ('error' in change) {
      form.setError('quantity', { message: change.error }, { shouldFocus: true });
      return;
    }
    const locationName = locations.find((l) => l.value === location_id)?.label ?? 'La sede';
    adjust.mutate(
      { variant_id: item.variant.id, location_id, quantity: change.delta, reason },
      {
        onSuccess: (movement) => {
          const description = adjustedMessage({
            item,
            delta: change.delta,
            balanceAfter: movement.balance_after,
            locationName,
          });
          toast({ tone: 'success', title: 'Ajuste registrado', description });
          form.reset({ item: null, location_id, direction: '', quantity: '', reason: '' });
        },
        onError: (error) => {
          if (error instanceof ApiError && error.code === 'insufficient_stock') {
            const message = insufficientStockMessage(item, locationName);
            form.setError('quantity', { message }, { shouldFocus: true });
          } else handleError(error);
        },
      },
    );
  });

  return { form, formError, submit, current, isPending: adjust.isPending };
}
