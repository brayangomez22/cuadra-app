import { zodResolver } from '@hookform/resolvers/zod';
import { useForm, useWatch } from 'react-hook-form';
import { toDecimalInput } from '@/shared/format';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import type { Product, ProductVariant, UnitOfMeasure, VariantUnit } from '../../../api/types';
import { useSaveVariantUnit } from '../../../api/variantUnits';
import {
  readFactor,
  variantUnitSchema,
  type VariantUnitFormInput,
  type VariantUnitFormValues,
} from '../../../model/variants/variantUnitSchema';
import { suggestedUnitPrice } from '../../../model/variants/variants';

/** The unit of a variant the dialog edits; without unit, it adds one. */
export type UnitTarget = { product: Product; variant: ProductVariant; unit?: VariantUnit };

// Error codes of PUT …/units/{unitId} (api/openapi.yaml) that belong to one field.
const fieldByErrorCode = {
  unit_not_found: 'unitId',
  alternate_unit_is_base: 'unitId',
  invalid_unit_factor: 'factor',
  invalid_price: 'price',
  invalid_barcode: 'barcode',
  barcode_taken: 'barcode',
} as const;

type Input = { target: UnitTarget; units: UnitOfMeasure[]; onSaved: () => void };

/** The form of an alternate unit: its fields, the price it suggests and the save. */
export function useUnitForm({ target: { product, variant, unit }, units, onSaved }: Input) {
  const form = useForm<VariantUnitFormInput, unknown, VariantUnitFormValues>({
    resolver: zodResolver(variantUnitSchema),
    defaultValues: {
      unitId: unit?.unit.id ?? '',
      factor: unit ? toDecimalInput(unit.factor) : '',
      price: unit ? toDecimalInput(unit.price) : '',
      barcode: unit?.barcode ?? '',
    },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fieldByErrorCode);
  const saveUnit = useSaveVariantUnit();
  const factor = readFactor(useWatch({ control: form.control, name: 'factor' }));
  const suggested = factor && suggestedUnitPrice(variant.price, factor);

  async function save(values: VariantUnitFormValues) {
    clearFormError();
    const chosen = units.find((u) => u.id === values.unitId) ?? { id: values.unitId, code: '', name: '' };
    try {
      await saveUnit.mutateAsync({
        productId: product.id,
        variantId: variant.id,
        unit: chosen,
        body: { factor: values.factor, price: values.price, ...(values.barcode && { barcode: values.barcode }) },
      });
      onSaved();
    } catch (error) {
      handleError(error);
    }
  }

  return {
    form,
    formError,
    factor,
    suggested,
    base: product.unit.code,
    basePrice: variant.price,
    submit: form.handleSubmit(save),
  };
}

export type UnitForm = ReturnType<typeof useUnitForm>;
