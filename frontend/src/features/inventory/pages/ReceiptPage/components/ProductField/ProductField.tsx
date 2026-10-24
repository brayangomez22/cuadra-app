import { Controller, useFormContext } from 'react-hook-form';
import { ProductPicker, type Sellable } from '@/features/catalog';
import type { ReceiptInput } from '../../../../model/receiptSchema';

/** The product (a variant, or one of its units by barcode); once picked, the quantity is next. */
export function ProductField({ onChange }: { onChange: (item: Sellable | null) => void }) {
  const { control, setFocus } = useFormContext<ReceiptInput>();
  return (
    <Controller
      control={control}
      name="item"
      render={({ field, fieldState }) => (
        <ProductPicker
          label="Producto"
          value={field.value ?? null}
          onChange={onChange}
          onPicked={() => setFocus('quantity')}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}
