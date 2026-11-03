import { z } from 'zod';
import type { components } from '@/shared/api/schema';
import { discountPercentText, readMaxDiscount } from './maxDiscount';

export const salesRulesSchema = z.object({
  maxDiscount: z
    .string()
    .superRefine((text, ctx) => {
      const read = readMaxDiscount(text);
      if (read.error) ctx.addIssue({ code: 'custom', message: read.error });
    })
    .transform((text) => readMaxDiscount(text).value ?? text),
  requireCashSession: z.boolean(),
});

export type SalesRulesInput = z.input<typeof salesRulesSchema>;
export type SalesRulesValues = z.output<typeof salesRulesSchema>;

type SalesSettings = components['schemas']['SalesSettings'];

/** The form's values for the tenant's rules: the limit in percent. */
export function salesRulesInput(rules: SalesSettings): SalesRulesInput {
  return { maxDiscount: discountPercentText(rules.max_discount), requireCashSession: rules.require_cash_session };
}

/** The body of PATCH /sales/settings for the form's values. */
export function salesRulesBody(values: SalesRulesValues): components['schemas']['SalesSettingsUpdate'] {
  return { max_discount: values.maxDiscount, require_cash_session: values.requireCashSession };
}
