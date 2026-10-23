import { z } from 'zod';
import { decimalField } from '@/shared/forms/decimalField';
import { parseDecimalInput } from '@/shared/format';
import { toUnits } from '@/shared/money/decimal';

/** An alternate unit of a variant: how many base units it holds and its price. */
export const variantUnitSchema = z.object({
  unitId: z.string().min(1, 'Elija la unidad'),
  factor: decimalField('positive', {
    empty: 'Escriba cuántas trae',
    belowMin: 'Debe traer más de cero',
  }),
  price: decimalField('positive', {
    empty: 'Escriba el precio',
    belowMin: 'El precio debe ser mayor que cero',
  }),
  barcode: z.string().trim().max(64, 'El código de barras puede tener máximo 64 caracteres'),
});

export type VariantUnitFormInput = z.input<typeof variantUnitSchema>;
export type VariantUnitFormValues = z.output<typeof variantUnitSchema>;

/** The factor typed, as the API's decimal, when it is a positive one. */
export function readFactor(text: string): string | undefined {
  const parsed = parseDecimalInput(text);
  return parsed.ok && toUnits(parsed.value) > 0n ? parsed.value : undefined;
}
