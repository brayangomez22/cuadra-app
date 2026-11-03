import { z } from 'zod';
import type { Sellable } from '@/features/catalog';
import { decimalField } from '@/shared/forms/decimalField';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

// Same limits as TransferInput in api/openapi.yaml.
export const transferSchema = z
  .object({
    item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
    from_location_id: z.string().min(1, 'Elija la sede de origen'),
    to_location_id: z.string().min(1, 'Elija la sede de destino'),
    quantity: decimalField('positive', {
      empty: 'Escriba la cantidad',
      belowMin: 'La cantidad debe ser mayor que cero',
    }),
  })
  .refine((values) => values.from_location_id !== values.to_location_id, {
    path: ['to_location_id'],
    message: 'La sede de destino debe ser distinta de la de origen',
  });

export type TransferFormInput = z.input<typeof transferSchema>;
export type TransferValues = z.output<typeof transferSchema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
export const transferErrorFields: FieldsByErrorCode<TransferFormInput> = {
  variant_not_found: 'item',
  location_not_found: ['from_location_id', 'to_location_id'],
  location_inactive: ['from_location_id', 'to_location_id'],
  invalid_quantity: 'quantity',
  same_location_transfer: 'to_location_id',
};
