import { z } from 'zod';
import { isNegative, isZero, parseMoneyInput } from '@/shared/format';

/**
 * An amount of pesos as a cashier types it ("150.000", "12500,50"): the form
 * gets the API's decimal string. min "positive" rejects zero.
 */
export function moneyField(min: 'positive' | 'nonnegative', empty: string) {
  return z
    .string()
    .superRefine((text, ctx) => {
      if (text.trim() === '') {
        ctx.addIssue({ code: 'custom', message: empty });
        return;
      }
      const parsed = parseMoneyInput(text);
      if (!parsed.ok || /\.\d{3,}$/.test(parsed.value)) {
        ctx.addIssue({
          code: 'custom',
          message: 'Escriba un valor, por ejemplo 50.000 o 50000',
        });
        return;
      }
      if (isNegative(parsed.value) || (min === 'positive' && isZero(parsed.value))) {
        ctx.addIssue({
          code: 'custom',
          message: min === 'positive' ? 'El valor debe ser mayor que cero' : 'El valor no puede ser negativo',
        });
      }
    })
    .transform((text) => {
      const parsed = parseMoneyInput(text);
      return parsed.ok ? parsed.value : text;
    });
}
