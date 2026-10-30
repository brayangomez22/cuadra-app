import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { variantName, type Sellable } from '@/features/catalog';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { formatQuantity } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useRecordCountEntry } from '../../../api/countEntries';
import {
  countEntryErrorFields,
  countEntrySchema,
  toCountEntryInput,
  type CountEntryFormInput,
  type CountEntryValues,
} from '../../../model/stockCount/countEntrySchema';

const empty: CountEntryFormInput = { item: null, quantity: '', unit_id: '' };

/** What the user counted of a product; after recording it the form is ready for the next one. */
export function useCountEntryForm(countId: string) {
  const { toast } = useToast();
  const record = useRecordCountEntry(countId);
  const form = useForm<CountEntryFormInput, unknown, CountEntryValues>({
    resolver: zodResolver(countEntrySchema),
    defaultValues: empty,
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, countEntryErrorFields);

  function choose(item: Sellable | null) {
    form.setValue('item', item, { shouldValidate: form.getFieldState('item').invalid });
    // A unit's barcode (a box) brings that unit: the person counts boxes.
    form.setValue('unit_id', item?.unit?.unit.id ?? '');
  }

  const chooseUnit = (_: Sellable | null, unitId: string) => form.setValue('unit_id', unitId);

  const submit = form.handleSubmit(async (values) => {
    clearFormError();
    try {
      await record.mutateAsync(toCountEntryInput(values));
      const { item } = values;
      const name = variantName(item.product.name, item.variant.label);
      const unit = item.unit?.unit.code ?? item.product.unit.code;
      toast({
        tone: 'success',
        title: 'Conteo registrado',
        description: `${formatQuantity(values.quantity)} ${unit} de ${name}`,
      });
      form.reset(empty);
      // The next scan of a barcode reader has to land on the product.
      requestAnimationFrame(() => form.setFocus('item'));
    } catch (error) {
      handleError(error);
    }
  });

  return { form, formError, choose, chooseUnit, submit, isPending: record.isPending };
}
