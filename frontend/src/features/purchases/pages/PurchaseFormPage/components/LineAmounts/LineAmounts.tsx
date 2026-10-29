import { useFormContext, useWatch } from 'react-hook-form';
import { taxRateOptions } from '@/features/catalog';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';
import { lineCostHint } from '../../../../model/purchaseLines/lineCostHint';
import { lineReceived } from '../../../../model/purchaseLines/lineReceived';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';
import { LineUnitField } from '../LineUnitField/LineUnitField';
import styles from './LineAmounts.module.scss';

// A line without a product offers the general rate until it takes the product's.
const DEFAULT_RATE = '0.19';

/** The unit (when the variant has others), the quantity, the cost without VAT and the VAT rate of a line. */
export function LineAmounts({ index }: { index: number }) {
  const { control, register } = useFormContext<PurchaseInputValues>();
  const line = useWatch({ control, name: `lines.${index}` });
  const errorOf = useFieldError(control);
  const received = lineReceived({ item: line.item, unitId: line.unit_id, quantity: line.quantity });
  return (
    <div className={styles.root}>
      <LineUnitField index={index} />
      <Field label="Cantidad" hint={received ?? undefined} error={errorOf(`lines.${index}.quantity`)}>
        <Input inputMode="decimal" align="end" autoComplete="off" {...register(`lines.${index}.quantity`)} />
      </Field>
      <Field
        label="Costo unitario"
        hint={lineCostHint(line.item, line.unit_id)}
        error={errorOf(`lines.${index}.unit_cost`)}
      >
        <Input inputMode="decimal" align="end" autoComplete="off" {...register(`lines.${index}.unit_cost`)} />
      </Field>
      <Field label="IVA" error={errorOf(`lines.${index}.tax_rate`)}>
        <SelectController
          control={control}
          name={`lines.${index}.tax_rate`}
          options={taxRateOptions(line.tax_rate || DEFAULT_RATE)}
          placeholder="Tasa"
        />
      </Field>
    </div>
  );
}
