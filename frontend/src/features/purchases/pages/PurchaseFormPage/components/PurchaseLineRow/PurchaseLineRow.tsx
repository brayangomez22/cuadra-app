import { Controller, useFormContext } from 'react-hook-form';
import { ProductPicker, type Sellable } from '@/features/catalog';
import { IconButton, IconTrash } from '@/shared/ui';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';
import { LineAmounts } from '../LineAmounts/LineAmounts';
import styles from './PurchaseLineRow.module.scss';

type Props = {
  index: number;
  /** Absent on the only line: a purchase has one at least. */
  onRemove?: () => void;
};

/**
 * A line of the invoice: the product (a variant, or one of its units by
 * barcode), its unit of purchase, quantity, cost and VAT. Picking the product
 * takes its unit and its VAT rate, and moves on to the quantity.
 */
export function PurchaseLineRow({ index, onRemove }: Props) {
  const { control, setValue, setFocus } = useFormContext<PurchaseInputValues>();
  const n = String(index + 1);
  const choose = (item: Sellable | null) => {
    setValue(`lines.${index}.unit_id`, item?.unit?.unit.id ?? '');
    setValue(`lines.${index}.tax_rate`, item?.product.tax_rate ?? '');
  };
  return (
    <fieldset className={styles.root}>
      <legend className={styles.legend}>Línea {n}</legend>
      {onRemove && (
        <IconButton className={styles.remove} label={`Quitar la línea ${n}`} icon={<IconTrash />} onClick={onRemove} />
      )}
      <Controller
        control={control}
        name={`lines.${index}.item`}
        render={({ field, fieldState }) => (
          <ProductPicker
            label="Producto"
            value={field.value ?? null}
            onChange={(item) => {
              field.onChange(item);
              choose(item);
            }}
            onPicked={() => setFocus(`lines.${index}.quantity`)}
            error={fieldState.error?.message}
          />
        )}
      />
      <LineAmounts index={index} />
    </fieldset>
  );
}
