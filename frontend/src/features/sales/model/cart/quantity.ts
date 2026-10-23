import { parseDecimalInput } from '@/shared/format';
import { toUnits } from '@/shared/money/decimal';

/** The quantity typed, as a decimal of the API ("2.5"), or why it is not one. */
export function readQuantity(text: string): { value?: string; error?: string } {
  if (text.trim() === '') return { error: 'Escriba la cantidad' };
  const parsed = parseDecimalInput(text);
  if (!parsed.ok) {
    const messages = {
      invalid: 'Escriba un número, por ejemplo 2 o 2,5',
      thousands: `Escriba la cantidad sin puntos de miles: ${text.trim().replaceAll('.', '')}`,
      decimals: 'Use máximo 4 decimales',
      too_large: 'La cantidad es demasiado grande',
    };
    return { error: messages[parsed.reason] };
  }
  if (toUnits(parsed.value) <= 0n) return { error: 'La cantidad debe ser mayor que cero' };
  return { value: parsed.value };
}
