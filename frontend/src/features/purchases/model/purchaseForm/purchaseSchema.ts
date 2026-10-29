import { z } from 'zod';
import type { Sellable } from '@/features/catalog';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { decimalField } from '@/shared/forms/decimalField';

// Same limits as PurchaseInput in api/openapi.yaml; the rest (a date after
// today, a due date before the invoice) are the API's and show on their field.
const lineSchema = z.object({
  item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
  /** An alternate unit of the variant (docs/decisiones.md, T26); empty is the product's unit. */
  unit_id: z.string(),
  quantity: decimalField('positive', {
    empty: 'Escriba la cantidad',
    belowMin: 'La cantidad debe ser mayor que cero',
  }),
  unit_cost: decimalField('nonnegative', {
    empty: 'Escriba el costo unitario',
    belowMin: 'El costo no puede ser negativo',
  }),
  tax_rate: z.string(),
});

export const purchaseSchema = z.object({
  supplier_id: z.string().min(1, 'Elija el proveedor'),
  location_id: z.string().min(1, 'Elija la sede'),
  invoice_number: z
    .string()
    .trim()
    .min(1, 'Escriba el número de la factura')
    .max(50, 'El número puede tener máximo 50 caracteres'),
  issued_on: z.string().min(1, 'Escriba la fecha de la factura'),
  payment: z.enum(['cash', 'credit']),
  due_on: z.string(),
  notes: z.string().max(500, 'Las notas pueden tener máximo 500 caracteres'),
  lines: z.array(lineSchema).min(1, 'Agregue al menos una línea').max(200, 'Una compra tiene máximo 200 líneas'),
});

export type PurchaseInputValues = z.input<typeof purchaseSchema>;
export type PurchaseValues = z.output<typeof purchaseSchema>;
export type PurchaseLineValues = PurchaseInputValues['lines'][number];

// Error codes of POST /purchases (api/openapi.yaml) that belong to a field of the invoice.
export const purchaseErrorFields: FieldsByErrorCode<PurchaseInputValues> = {
  supplier_not_found: 'supplier_id',
  supplier_inactive: 'supplier_id',
  location_not_found: 'location_id',
  location_inactive: 'location_id',
  invalid_invoice_number: 'invoice_number',
  purchase_invoice_taken: 'invoice_number',
  invoice_date_in_future: 'issued_on',
  due_before_issue: 'due_on',
  invalid_purchase_notes: 'notes',
};

type LineField = 'item' | 'unit_id' | 'quantity' | 'unit_cost' | 'tax_rate';

// The field of a line each error with details.line belongs to; any other is the product's.
const lineErrorFields: Partial<Record<string, LineField>> = {
  unit_not_available: 'unit_id',
  invalid_quantity: 'quantity',
  quantity_precision: 'quantity',
  invalid_unit_cost: 'unit_cost',
  invalid_tax_rate: 'tax_rate',
};

/** The field of line n where an error of the API shows: "lines.2.quantity". */
export function lineErrorField(code: string, line: number): `lines.${number}.${LineField}` {
  return `lines.${line}.${lineErrorFields[code] ?? 'item'}`;
}
