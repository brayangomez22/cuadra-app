import { useFormContext, useWatch } from 'react-hook-form';
import type { Sellable } from '@/features/catalog';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';

type Fields = { item: Sellable | null; quantity: string };

/** The quantity of an adjustment or a transfer, always in the product's base unit: their API takes no other. */
export function QuantityField() {
  const { control, register } = useFormContext<Fields>();
  const item = useWatch({ control, name: 'item' });
  const errorOf = useFieldError(control, 'quantity');
  return (
    <Field label="Cantidad" hint={item ? `En ${item.product.unit.code}.` : undefined} error={errorOf('quantity')}>
      <Input inputMode="decimal" align="end" autoComplete="off" {...register('quantity')} />
    </Field>
  );
}
