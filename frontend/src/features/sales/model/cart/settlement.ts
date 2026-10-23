import { fromUnits, toUnits } from '@/shared/money/decimal';
import type { PaymentInput } from '../paymentMethods';

export type Settlement = {
  paid: string;
  /** What is still to be paid. */
  missing: string;
  /** The change in cash. */
  change: string;
  /** insufficient: they pay less than the total; non_cash_overpayment: only cash gives change. */
  error?: 'insufficient' | 'non_cash_overpayment';
};

/** The payments against the total, with the same rules as the API. */
export function settle(total: string, payments: PaymentInput[]): Settlement {
  const due = toUnits(total);
  let paid = 0n;
  let nonCash = 0n;
  for (const p of payments) {
    paid += toUnits(p.amount);
    if (p.method !== 'cash') nonCash += toUnits(p.amount);
  }
  const settlement: Settlement = {
    paid: fromUnits(paid),
    missing: fromUnits(paid < due ? due - paid : 0n),
    change: fromUnits(paid > due ? paid - due : 0n),
  };
  if (paid < due) settlement.error = 'insufficient';
  else if (nonCash > due) settlement.error = 'non_cash_overpayment';
  return settlement;
}
