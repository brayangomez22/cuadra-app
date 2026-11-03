import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { Sellable } from '@/features/catalog';
import { ApiError } from '@/shared/api/errors';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useTransferStock } from '../../../api/adjustments';
import { insufficientStockMessage, transferredMessage } from '../../../model/stockChange/stockChangeMessages';
import {
  transferErrorFields,
  transferSchema,
  type TransferFormInput,
  type TransferValues,
} from '../../../model/stockChange/transferSchema';

export type TransferSetup = {
  locations: { value: string; label: string }[];
  initialItem: Sellable | null;
  initialLocation: string;
};

/** The transfer form; after one it is ready for the next product between the same two locations. */
export function useTransferForm({ locations, initialItem, initialLocation }: TransferSetup) {
  const { toast } = useToast();
  const transfer = useTransferStock();
  const form = useForm<TransferFormInput, unknown, TransferValues>({
    resolver: zodResolver(transferSchema),
    defaultValues: { item: initialItem, from_location_id: initialLocation, to_location_id: '', quantity: '' },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, transferErrorFields);
  const nameOf = (id: string) => locations.find((l) => l.value === id)?.label ?? 'La sede';

  const submit = form.handleSubmit(({ item, from_location_id, to_location_id, quantity }) => {
    clearFormError();
    transfer.mutate(
      { variant_id: item.variant.id, from_location_id, to_location_id, quantity },
      {
        onSuccess: ({ out, in: entered }) => {
          const from = { name: nameOf(from_location_id), balance: out.balance_after };
          const to = { name: nameOf(to_location_id), balance: entered.balance_after };
          toast({
            tone: 'success',
            title: 'Traslado registrado',
            description: transferredMessage({ item, quantity, from, to }),
          });
          form.reset({ item: null, from_location_id, to_location_id, quantity: '' });
        },
        onError: (error) => {
          if (error instanceof ApiError && error.code === 'insufficient_stock') {
            const message = insufficientStockMessage(item, nameOf(from_location_id));
            form.setError('quantity', { message }, { shouldFocus: true });
          } else handleError(error);
        },
      },
    );
  });

  return { form, formError, submit, nameOf, isPending: transfer.isPending };
}
