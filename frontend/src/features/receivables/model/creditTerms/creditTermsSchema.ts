import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { moneyField } from '@/shared/forms/moneyField';
import { toDecimalInput } from '@/shared/format';
import type { CustomerCredit } from '../../api/types';

// Same range as CreditTermsInput in api/openapi.yaml.
const MAX_TERM_DAYS = 365;

/** The credit limit and term of a customer as typed; the form gets CreditTermsInput. */
export const creditTermsSchema = z.object({
  credit_limit: moneyField('nonnegative', 'Escriba el cupo, aunque sea 0'),
  term_days: z
    .string()
    .regex(/^\s*\d+\s*$/, 'Escriba el plazo en días enteros')
    .transform(Number)
    .refine((days) => days <= MAX_TERM_DAYS, `El plazo va de 0 a ${String(MAX_TERM_DAYS)} días`),
});

export type CreditTermsValues = z.input<typeof creditTermsSchema>;

/** The credit's limit and term as the form shows them: "1500000.00" → "1500000". */
export function creditTermsDefaults(credit: Pick<CustomerCredit, 'credit_limit' | 'term_days'>): CreditTermsValues {
  return {
    credit_limit: toDecimalInput(credit.credit_limit),
    term_days: String(credit.term_days),
  };
}

// Error codes of the API (api/openapi.yaml) that belong to one field.
export const creditTermsErrorFields: FieldsByErrorCode<CreditTermsValues> = {
  invalid_credit_limit: 'credit_limit',
  invalid_credit_term: 'term_days',
};
