import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import type { Sellable } from '@/features/catalog';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { toDecimalInput } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useReceiveGoods } from '../../../api/receipts';
import { receiptErrorFields, receiptSchema, type ReceiptInput, type ReceiptValues } from '../../../model/receiptSchema';
import { receivedMessage } from '../../../model/receivedMessage';
import { suggestedCost } from '../../../model/suggestedCost';

export type ReceiptSetup = {
  locations: { value: string; label: string }[];
  initialItem: Sellable | null;
  initialLocation: string;
};

/** The entry form: choosing the product and the unit proposes the cost; after an entry it is ready for the next one. */
export function useReceiptForm({ locations, initialItem, initialLocation }: ReceiptSetup) {
  const { toast } = useToast();
  const receive = useReceiveGoods();
  const form = useForm<ReceiptInput, unknown, ReceiptValues>({
    resolver: zodResolver(receiptSchema),
    defaultValues: {
      item: initialItem,
      location_id: initialLocation,
      quantity: '',
      unit_cost: initialItem?.variant.cost ? toDecimalInput(initialItem.variant.cost) : '',
      unit_id: '',
    },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, receiptErrorFields);
  const { setFocus, setValue, resetField } = form;

  // Coming from the kardex, the product is already there: the quantity is next.
  useEffect(() => {
    if (initialItem) setFocus('quantity');
  }, [initialItem, setFocus]);

  function chooseUnit(item: Sellable | null, unitId: string) {
    setValue('unit_id', unitId);
    const cost = item && suggestedCost(item, unitId);
    if (cost) setValue('unit_cost', toDecimalInput(cost));
  }

  function choose(item: Sellable | null) {
    setValue('item', item, { shouldValidate: form.getFieldState('item').invalid });
    // A unit's barcode brings that unit, with the module on or off: the API
    // takes the units already defined.
    chooseUnit(item, item?.unit?.unit.id ?? '');
  }

  const submit = form.handleSubmit(({ item, ...values }) => {
    clearFormError();
    const unitId = values.unit_id;
    const body = { variant_id: item.variant.id, ...values, unit_id: unitId || undefined };
    receive.mutate(body, {
      onSuccess: (movement) => {
        const locationName = locations.find((l) => l.value === values.location_id)?.label ?? 'La sede';
        const description = receivedMessage({ item, unitId, quantity: values.quantity, movement, locationName });
        toast({ tone: 'success', title: 'Entrada registrada', description });
        // Ready for the next product of the same delivery, in the same location.
        resetField('item', { defaultValue: null });
        resetField('quantity', { defaultValue: '' });
        resetField('unit_cost', { defaultValue: '' });
        resetField('unit_id', { defaultValue: '' });
      },
      onError: handleError,
    });
  });

  return { form, formError, choose, chooseUnit, submit, isPending: receive.isPending };
}
