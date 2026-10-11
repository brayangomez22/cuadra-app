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
 * A text field with an amount or a quantity (docs/decisiones.md): the person
 * types "12500,50" and the form gets "12500.50", the API's decimal string.
 * min "positive" rejects zero and negatives; "nonnegative" only negatives.
 */
export function decimalField(min: 'positive' | 'nonnegative', messages: Messages) {
  return z
    .string()
    .superRefine((text, ctx) => {
      if (text.trim() === '') {
        ctx.addIssue({ code: 'custom', message: messages.empty });
        return;
      }
      const parsed = parseDecimalInput(text);
      if (!parsed.ok) {
        ctx.addIssue({ code: 'custom', message: reasons[parsed.reason](text) });
        return;
      }
      if (isNegative(parsed.value) || (min === 'positive' && isZero(parsed.value))) {
        ctx.addIssue({ code: 'custom', message: messages.belowMin });
      }
    })
    .transform((text) => {
      const parsed = parseDecimalInput(text);
      return parsed.ok ? parsed.value : text;
    });
}
