import { z } from 'zod';
import type { Sellable } from '@/features/catalog';
import { decimalField } from '@/shared/forms/decimalField';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { isZero } from '@/shared/format';
import type { AdjustmentMode } from './stockChange';

export const MODES: { value: AdjustmentMode; label: string }[] = [
  { value: 'out', label: 'Sale' },
  { value: 'in', label: 'Entra' },
  { value: 'count', label: 'Contar lo que hay' },
];

/** The usual reasons, as shortcuts: the reason itself is free text. */
export const REASONS: Record<AdjustmentMode, string[]> = {
  out: ['Rotura', 'Pérdida', 'Vencido', 'Consumo interno'],
  in: ['Mercancía encontrada', 'Corrección de un error'],
  count: ['Diferencia al contar'],
};

const quantityMessages = { empty: 'Escriba la cantidad', belowMin: 'La cantidad debe ser mayor que cero' };

// Same limits as AdjustmentInput in api/openapi.yaml. There is no default
// direction: adding what broke instead of removing it doubles the mistake.
// A count may be zero (nothing left); what leaves or enters may not.
export const adjustmentSchema = z
  .object({
    item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
    location_id: z.string().min(1, 'Elija la sede'),
    direction: z.custom<AdjustmentMode | ''>().refine(Boolean, 'Elija si sale, entra o cuenta lo que hay'),
    quantity: decimalField('nonnegative', quantityMessages),
    reason: z.string().trim().min(1, 'Escriba el motivo').max(500, 'Use máximo 500 caracteres'),
  })
  .superRefine(({ direction, quantity }, ctx) => {
    if (direction !== 'count' && isZero(quantity)) {
      ctx.addIssue({ code: 'custom', path: ['quantity'], message: quantityMessages.belowMin });
    }
  });

export type AdjustmentFormInput = z.input<typeof adjustmentSchema>;
export type AdjustmentValues = z.output<typeof adjustmentSchema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
export const adjustmentErrorFields: FieldsByErrorCode<AdjustmentFormInput> = {
  variant_not_found: 'item',
  location_not_found: 'location_id',
  location_inactive: 'location_id',
  invalid_quantity: 'quantity',
  adjustment_reason_required: 'reason',
  adjustment_reason_too_long: 'reason',
};
