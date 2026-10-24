import { z } from 'zod';
import type { Sellable } from '@/features/catalog';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { decimalField } from '@/shared/forms/decimalField';

// Same limits as ReceiptInput in api/openapi.yaml.
export const receiptSchema = z.object({
  item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
  location_id: z.string().min(1, 'Elija la sede'),
  quantity: decimalField('positive', {
    empty: 'Escriba la cantidad',
    belowMin: 'La cantidad debe ser mayor que cero',
  }),
  unit_cost: decimalField('nonnegative', {
    empty: 'Escriba el costo unitario',
    belowMin: 'El costo no puede ser negativo',
  }),
  /** An alternate unit of the variant (docs/decisiones.md, T26); empty is the product's unit. */
  unit_id: z.string(),
});

export type ReceiptInput = z.input<typeof receiptSchema>;
export type ReceiptValues = z.output<typeof receiptSchema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
export const receiptErrorFields: FieldsByErrorCode<ReceiptInput> = {
  product_not_found: 'item',
  location_not_found: 'location_id',
  location_inactive: 'location_id',
  invalid_quantity: 'quantity',
  invalid_unit_cost: 'unit_cost',
  unit_not_available: 'unit_id',
  quantity_precision: 'quantity',
};
