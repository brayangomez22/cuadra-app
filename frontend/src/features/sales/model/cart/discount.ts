import { parseDecimalInput } from '@/shared/format';
import { fromUnitsTrimmed, toUnits } from '@/shared/money/decimal';
import type { components } from '@/shared/api/schema';

export type DiscountLimit = components['schemas']['DiscountLimit'];

type Read = { fraction?: bigint; error?: string };

/**
 * The percent typed as a fraction of the line's gross amount ("10" → 1000n
 * units, 0.10), checked against the user's limit.
 */
export function readDiscount(text: string, limit: DiscountLimit): Read {
  if (text.trim() === '') return { fraction: 0n };
  const parsed = parseDecimalInput(text);
  if (!parsed.ok) {
    if (parsed.reason === 'decimals') return { error: 'Use máximo 2 decimales' };
    return { error: 'Escriba un porcentaje, por ejemplo 5 o 2,5' };
  }
  const percent = toUnits(parsed.value);
  // Two decimals of a percent are four of a fraction: the division is exact.
  if (percent % 100n !== 0n) return { error: 'Use máximo 2 decimales' };
  const fraction = percent / 100n;
  if (fraction < 0n || fraction > toUnits('1')) return { error: 'El descuento va de 0 a 100 %' };
  if (fraction === 0n) return { fraction };
  return withinLimit(fraction, limit);
}

function withinLimit(fraction: bigint, limit: DiscountLimit): Read {
  if (limit.unlimited) return { fraction };
  const max = toUnits(limit.max);
  if (max === 0n) return { error: 'No puede dar descuentos' };
  if (fraction > max) return { error: `Máximo ${fromUnitsTrimmed(max * 100n).replace('.', ',')} %` };
  return { fraction };
}
