import { useFormContext, useWatch } from 'react-hook-form';
import { Field, Input } from '@/shared/ui';
import { useFieldError } from '@/shared/forms/useFieldError';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import type { ProductFormInput } from '../../../../model/productSchema';
import { taxRateOptions } from '../../../../model/taxRates';
import { FormGrid, FormSection } from '../FormSection/FormSection';

type Props = {
  /** With the variant matrix, the cost and price are what the variants without theirs take. */
  matrix: boolean;
  /** An inactive product's price cannot change. */
  inactive: boolean;
};

function priceHint({ matrix, inactive }: Props) {
  if (inactive) return 'Active el producto para cambiar el precio.';
  return matrix ? 'Lo usan las variantes que no tengan el suyo.' : 'Use coma para los centavos: 12500,50.';
}

/** "Precio": cost, price before IVA and the IVA rate. */
export function PriceSection({ matrix, inactive }: Props) {
  const { register, control } = useFormContext<ProductFormInput>();
  const error = useFieldError(control);
  const taxRate = useWatch({ control, name: 'tax_rate' });
  return (
    <FormSection title="Precio">
      <FormGrid>
        <Field
          label="Costo"
          hint={matrix ? 'Sin IVA. Lo usan las variantes que no tengan el suyo.' : 'Lo que le cuesta, sin IVA.'}
          error={error('cost')}
        >
          <Input inputMode="decimal" align="end" autoComplete="off" {...register('cost')} />
        </Field>
        <Field label="Precio de venta (sin IVA)" hint={priceHint({ matrix, inactive })} error={error('price')}>
          <Input inputMode="decimal" align="end" autoComplete="off" disabled={inactive} {...register('price')} />
        </Field>
        <Field label="IVA" error={error('tax_rate')}>
          <SelectController control={control} name="tax_rate" options={taxRateOptions(taxRate)} />
        </Field>
      </FormGrid>
    </FormSection>
  );
}
