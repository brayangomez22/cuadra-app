import { z } from 'zod';
import { decimalField } from '@/shared/forms/decimalField';

// Same limits as ProductInput in api/openapi.yaml. The domain rules (SKU and
// barcode format, uniqueness) are checked by the API and shown on the field.
export const productSchema = z.object({
  sku: z.string().trim().min(1, 'Escriba el SKU').max(64, 'El SKU puede tener máximo 64 caracteres'),
  barcode: z.string().trim().max(64, 'El código de barras puede tener máximo 64 caracteres'),
  name: z
    .string()
    .trim()
    .min(1, 'Escriba el nombre del producto')
    .max(200, 'El nombre puede tener máximo 200 caracteres'),
  description: z.string().trim().max(2000, 'La descripción puede tener máximo 2000 caracteres'),
  // NO_CATEGORY when the product has none (Radix Select has no empty value).
  category_id: z.string(),
  // The unit's code (the deprecated base_unit of the API) until T24j moves
  // the form to base_unit_id and the tenant's units.
  base_unit: z.string().min(1),
  tax_rate: z.string(),
  cost: decimalField('nonnegative', {
    empty: 'Escriba el costo (0 si aún no lo sabe)',
    belowMin: 'El costo no puede ser negativo',
  }),
  price: decimalField('positive', {
    empty: 'Escriba el precio de venta',
    belowMin: 'El precio debe ser mayor que cero',
  }),
});

export type ProductFormInput = z.input<typeof productSchema>;
export type ProductFormValues = z.output<typeof productSchema>;

export const NO_CATEGORY = 'none';
