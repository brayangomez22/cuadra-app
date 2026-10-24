import { useFormContext } from 'react-hook-form';
import { Field, Input, Textarea } from '@/shared/ui';
import type { Category, UnitOfMeasure } from '../../../../api/types';
import { useFieldError } from '@/shared/forms/useFieldError';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { categoryOptions } from '../../../../model/productForm/categoryOptions';
import type { ProductFormInput } from '../../../../model/productForm/productSchema';
import { FormGrid, FormGridWide, FormSection } from '../FormSection/FormSection';

type Props = {
  categories: Category[];
  units: UnitOfMeasure[];
  /** With the variant matrix, the SKU is the base of the variants' and each one has its barcode. */
  matrix: boolean;
};

/** "Datos del producto": its codes, name, description, category and unit. */
export function GeneralSection({ categories, units, matrix }: Props) {
  const { register, control } = useFormContext<ProductFormInput>();
  const error = useFieldError(control);
  const skuHint = matrix
    ? 'La base del SKU de cada variante. Se guarda en mayúsculas.'
    : 'Su código interno. Se guarda en mayúsculas.';
  return (
    <FormSection title="Datos del producto">
      <FormGrid>
        <Field label="SKU" hint={skuHint} error={error('sku')}>
          <Input autoComplete="off" autoCapitalize="characters" spellCheck={false} {...register('sku')} />
        </Field>
        {!matrix && (
          <Field label="Código de barras" optional hint="Puede escanearlo aquí." error={error('barcode')}>
            <Input autoComplete="off" spellCheck={false} {...register('barcode')} />
          </Field>
        )}
        <FormGridWide>
          <Field label="Nombre" error={error('name')}>
            <Input autoComplete="off" {...register('name')} />
          </Field>
        </FormGridWide>
        <FormGridWide>
          <Field label="Descripción" optional error={error('description')}>
            <Textarea {...register('description')} />
          </Field>
        </FormGridWide>
        <Field label="Categoría" error={error('category_id')}>
          <SelectController control={control} name="category_id" options={categoryOptions(categories)} />
        </Field>
        <Field label="Unidad" hint="En la que cuenta y vende el producto." error={error('base_unit_id')}>
          <SelectController
            control={control}
            name="base_unit_id"
            options={units.map((unit) => ({ value: unit.id, label: `${unit.name} (${unit.code})` }))}
            placeholder="Elija la unidad"
          />
        </Field>
      </FormGrid>
    </FormSection>
  );
}
