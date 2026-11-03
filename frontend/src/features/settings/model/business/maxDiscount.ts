import { parseDecimalInput } from '@/shared/format';
import { fromUnitsTrimmed, toUnits } from '@/shared/money/decimal';

type Read = { value: string; error?: undefined } | { value?: undefined; error: string };

/**
 * The cashier's discount limit as the owner types it, in percent ("15"),
 * into the API's fraction ("0.15"). Empty is 0 %: no discount.
 */
export function readMaxDiscount(text: string): Read {
  if (text.trim() === '') return { value: '0' };
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
  return { value: fromUnitsTrimmed(fraction) };
}

/** The API's fraction ("0.025") as the percent of the field ("2,5"). */
export function discountPercentText(fraction: string): string {
  return fromUnitsTrimmed(toUnits(fraction) * 100n).replace('.', ',');
}
