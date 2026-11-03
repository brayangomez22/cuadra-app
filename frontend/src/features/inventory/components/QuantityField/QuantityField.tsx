import { useFormContext, useWatch } from 'react-hook-form';
import type { Sellable } from '@/features/catalog';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';

type Fields = { item: Sellable | null; quantity: string };

/**
 * The quantity of an adjustment or a transfer, always in the product's base
 * unit: their API takes no other. label changes for a count ("Cuántas hay").
 */
export function QuantityField({ label = 'Cantidad' }: { label?: string }) {
  const { control, register } = useFormContext<Fields>();
  const item = useWatch({ control, name: 'item' });
  const errorOf = useFieldError(control, 'quantity');
  return (
    <Field label={label} hint={item ? `En ${item.product.unit.code}.` : undefined} error={errorOf('quantity')}>
      <Input inputMode="decimal" align="end" autoComplete="off" {...register('quantity')} />
    </Field>
  );
}
