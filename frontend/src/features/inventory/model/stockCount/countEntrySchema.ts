import { z } from 'zod';
import type { Sellable } from '@/features/catalog';
import type { components } from '@/shared/api/schema';
import { decimalField } from '@/shared/forms/decimalField';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

// Same limits as CountEntryInput in api/openapi.yaml; zero is a count (an empty shelf).
export const countEntrySchema = z.object({
  item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
  quantity: decimalField('nonnegative', {
    empty: 'Escriba lo que contó',
    belowMin: 'La cantidad no puede ser negativa',
  }),
  /** An alternate unit of the variant; empty is the product's unit. */
  unit_id: z.string(),
});

export type CountEntryFormInput = z.input<typeof countEntrySchema>;
export type CountEntryValues = z.output<typeof countEntrySchema>;

/** The body of POST /stock-counts/{id}/entries. */
export function toCountEntryInput(values: CountEntryValues): components['schemas']['CountEntryInput'] {
  const body: components['schemas']['CountEntryInput'] = {
    variant_id: values.item.variant.id,
    quantity: values.quantity,
  };
  if (values.unit_id) body.unit_id = values.unit_id;
  return body;
}

// Error codes of the API (api/openapi.yaml) that belong to one field. A
// product outside a partial count goes above the form: the chosen product
// shows no error of its own.
export const countEntryErrorFields: FieldsByErrorCode<CountEntryFormInput> = {
  invalid_quantity: 'quantity',
  quantity_precision: 'quantity',
  unit_not_available: 'unit_id',
};
