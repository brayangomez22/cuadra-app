import { z } from 'zod';
import { isNegative, isZero, parseDecimalInput } from '@/shared/format';

type Messages = {
  /** The field is empty. */
  empty: string;
  /** The value is below the minimum. */
  belowMin: string;
};

const reasons = {
  invalid: () => 'Escriba un número, por ejemplo 12500 o 12500,50',
  thousands: (text: string) => `Escriba el valor sin puntos de miles: ${text.trim().replaceAll('.', '')}`,
  decimals: () => 'Use máximo 4 decimales',
  too_large: () => 'El valor es demasiado grande',
};

/**
 * Reads an amount or a quantity as the person types it ("12500,50") into the
 * API's decimal string ("12500.50"), or the message of what to fix. min
 * "positive" rejects zero and negatives, "nonnegative" only negatives and
 * "any" none.
 */
export function readDecimal(
  text: string,
  min: 'positive' | 'nonnegative' | 'any',
  messages: Messages,
): { value: string; error?: undefined } | { value?: undefined; error: string } {
  if (text.trim() === '') return { error: messages.empty };
  const parsed = parseDecimalInput(text);
  if (!parsed.ok) return { error: reasons[parsed.reason](text) };
  if ((min !== 'any' && isNegative(parsed.value)) || (min === 'positive' && isZero(parsed.value))) {
    return { error: messages.belowMin };
  }
  return { value: parsed.value };
}

/**
 * A text field with an amount or a quantity (docs/decisiones.md): the person
 * types "12500,50" and the form gets "12500.50", the API's decimal string.
 * min "positive" rejects zero and negatives; "nonnegative" only negatives.
 */
export function decimalField(min: 'positive' | 'nonnegative', messages: Messages) {
  return z
    .string()
    .superRefine((text, ctx) => {
      const read = readDecimal(text, min, messages);
      if (read.error) ctx.addIssue({ code: 'custom', message: read.error });
    })
    .transform((text) => readDecimal(text, min, messages).value ?? text);
}
