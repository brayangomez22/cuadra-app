import { useFormContext, useWatch } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';
import { receiptHints } from '../../../../model/receipt/receiptHints';
import type { ReceiptInput } from '../../../../model/receipt/receiptSchema';
import styles from './AmountFields.module.scss';

/** The quantity and the unit cost, both in the unit the goods arrive in. */
export function AmountFields() {
  const { control, register } = useFormContext<ReceiptInput>();
  const item = useWatch({ control, name: 'item' });
  const unitId = useWatch({ control, name: 'unit_id' });
  const errorOf = useFieldError(control);
  const hints = receiptHints(item ?? null, unitId);

  return (
    <div className={styles.root}>
      <Field label="Cantidad" hint={hints.quantity} error={errorOf('quantity')}>
        <Input inputMode="decimal" align="end" autoComplete="off" {...register('quantity')} />
      </Field>
      <Field label="Costo unitario" hint={hints.cost} error={errorOf('unit_cost')}>
        <Input inputMode="decimal" align="end" autoComplete="off" {...register('unit_cost')} />
      </Field>
    </div>
  );
}
