import { fromUnits, toUnits } from '@/shared/money/decimal';
import type { Receivable } from '../api/types';
import { dueState } from './dueState';

export type Overdue = { count: number; balance: string; oldestDays: number };

/**
 * What of the customer's open receivables is past due on today: how many,
 * how much and how many days the oldest one is late. Undefined when nothing
 * is overdue.
 */
export function overdueSummary(
  receivables: Pick<Receivable, 'balance' | 'due_date'>[],
  today: string,
): Overdue | undefined {
  let count = 0;
  let units = 0n;
  let oldestDays = 0;
  for (const receivable of receivables) {
    const due = dueState(receivable.due_date, today);
    if (due.state !== 'overdue') continue;
    count += 1;
    units += toUnits(receivable.balance);
    oldestDays = Math.max(oldestDays, due.days);
  }
  return count === 0 ? undefined : { count, balance: fromUnits(units), oldestDays };
}
