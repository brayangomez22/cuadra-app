import { Controller, useFormContext } from 'react-hook-form';
import { ProductPicker, type Sellable } from '@/features/catalog';

type Props = {
  /** Replaces setting the field, for a form that does more when the product changes. */
  onChange?: (item: Sellable | null) => void;
  onPicked?: () => void;
};

/** The `item` of a form that moves stock: a variant, or one of its units by barcode. */
export function ProductField({ onChange, onPicked }: Props) {
  const { control } = useFormContext<{ item: Sellable | null }>();
  return (
    <Controller
      control={control}
      name="item"
      render={({ field, fieldState }) => (
        <ProductPicker
          label="Producto"
          value={field.value ?? null}
          onChange={onChange ?? field.onChange}
          onPicked={onPicked}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}
