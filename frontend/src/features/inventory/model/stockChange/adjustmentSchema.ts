import { z } from 'zod';
import type { Sellable } from '@/features/catalog';
import { decimalField } from '@/shared/forms/decimalField';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import type { AdjustmentDirection } from './stockChange';

export const DIRECTIONS: { value: AdjustmentDirection; label: string }[] = [
  { value: 'out', label: 'Sacar de la existencia' },
  { value: 'in', label: 'Agregar a la existencia' },
];

/** The usual reasons, as shortcuts: the reason itself is free text. */
export const REASONS: Record<AdjustmentDirection, string[]> = {
  out: ['Rotura', 'Pérdida', 'Vencido', 'Consumo interno'],
  in: ['Mercancía encontrada', 'Corrección de un error'],
};

const quantityMessages = { empty: 'Escriba la cantidad', belowMin: 'La cantidad debe ser mayor que cero' };

// Same limits as AdjustmentInput in api/openapi.yaml. There is no default
// direction: adding what broke instead of removing it doubles the mistake.
export const adjustmentSchema = z.object({
  item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
  location_id: z.string().min(1, 'Elija la sede'),
  direction: z.custom<AdjustmentDirection | ''>().refine(Boolean, 'Elija si saca o agrega'),
  quantity: decimalField('positive', quantityMessages),
  reason: z.string().trim().min(1, 'Escriba el motivo').max(500, 'Use máximo 500 caracteres'),
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
