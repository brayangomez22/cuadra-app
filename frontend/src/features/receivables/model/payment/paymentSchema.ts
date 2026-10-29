import { z } from 'zod';
import { formatMoney } from '@/shared/format';
import { moneyField } from '@/shared/forms/moneyField';
import { toUnits } from '@/shared/money/decimal';
import type { CustomerPaymentMethod } from '../../api/types';

// moneyField leaves its text as is when it is not an amount; only an amount is compared.
const isAmount = (value: string) => /^\d+(\.\d{1,2})?$/.test(value);

/** A payment of a customer who owes balance: more than zero and never more than what they owe (as the API). */
export function paymentSchema(balance: string) {
  return z.object({
    method: z.custom<CustomerPaymentMethod>(),
    amount: moneyField('positive', 'Escriba el valor del abono').refine(
      (amount) => !isAmount(amount) || toUnits(amount) <= toUnits(balance),
      `El abono no puede pasar de lo que debe (${formatMoney(balance)})`,
    ),
  });
}

export type PaymentValues = z.input<ReturnType<typeof paymentSchema>>;
