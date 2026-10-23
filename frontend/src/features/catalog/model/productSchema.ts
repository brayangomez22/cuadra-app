import { z } from 'zod';
import { decimalField, readDecimal } from '@/shared/forms/decimalField';
import type { AttributeDefinition } from '../api/types';

export const NO_CATEGORY = 'none';
/** The value of a list attribute left empty (Radix Select has no empty value). */
export const NO_VALUE = '__none';

export const REQUIRED_ATTRIBUTE = 'Este dato es obligatorio';

// attributeField is one attribute's field, by its type. The text of a number
// becomes the API's decimal string ("2,5" → "2.5"); an empty optional value
// stays "" and is not sent.
function attributeField(definition: AttributeDefinition) {
  const { required } = definition;
  switch (definition.type) {
    case 'boolean':
      return z.boolean();
    case 'number':
      return z
        .string()
        .superRefine((text, ctx) => {
          if (text.trim() === '') {
            if (required) ctx.addIssue({ code: 'custom', message: REQUIRED_ATTRIBUTE });
            return;
          }
          const read = readDecimal(text, 'any', { empty: REQUIRED_ATTRIBUTE, belowMin: '' });
          if (read.error) ctx.addIssue({ code: 'custom', message: read.error });
        })
        .transform((text) =>
          text.trim() === '' ? '' : (readDecimal(text, 'any', { empty: '', belowMin: '' }).value ?? text),
        );
    case 'list':
      return z
        .string()
        .refine((value) => !required || (value !== '' && value !== NO_VALUE), REQUIRED_ATTRIBUTE)
        .transform((value) => (value === NO_VALUE ? '' : value));
    case 'text':
      return z
        .string()
        .trim()
        .max(500, 'Puede tener máximo 500 caracteres')
        .refine((value) => !required || value !== '', REQUIRED_ATTRIBUTE);
  }
}

// A variant of the matrix (module variants), as the person typed it. An empty
// SKU, cost or price takes the product's: the SKU base with the values
// ("CAM-M-AZUL"), and the cost and price above the table. They are checked
// when saving, against those.
const variantRow = z.object({
  /** The variant's id once it exists. */
  variantId: z.string().optional(),
  values: z.record(z.string(), z.string()),
  sku: z.string(),
  barcode: z.string(),
  cost: z.string(),
  price: z.string(),
  active: z.boolean(),
});

export type VariantRow = z.input<typeof variantRow>;

/**
 * The product form, with the tenant's attributes. Same limits as ProductInput
 * in api/openapi.yaml. The domain rules (SKU and barcode format, uniqueness)
 * are checked by the API and shown on the field.
 */
export function productSchema(definitions: AttributeDefinition[]) {
  return z.object({
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
    base_unit_id: z.string().min(1, 'Elija la unidad'),
    tax_rate: z.string(),
    cost: decimalField('nonnegative', {
      empty: 'Escriba el costo (0 si aún no lo sabe)',
      belowMin: 'El costo no puede ser negativo',
    }),
    price: decimalField('positive', {
      empty: 'Escriba el precio de venta',
      belowMin: 'El precio debe ser mayor que cero',
    }),
    attributes: z.object(Object.fromEntries(definitions.map((d) => [d.key, attributeField(d)]))),
    variants: z.array(variantRow),
  });
}

type Schema = ReturnType<typeof productSchema>;
export type ProductFormInput = z.input<Schema>;
export type ProductFormValues = z.output<Schema>;
